package main

import (
	"context"
	"fmt"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"darlinggo.co/pan"

	"github.com/dave/jennifer/jen"
	"impractical.co/clif"
)

var generateCmd = clif.Command{
	Name:                "generate",
	Description:         "generate the column mapping files",
	DetailedDescription: "pan generate finds all the struct types in the passed packages that implement the darlinggo.co/pan.SQLTableNamer interface and generates a companion type and constant for them. The companion types define a method for each struct field mapped to a SQL column, and return the SQL column name when called. darlinggo.co/pan.Flag values can be passed to quote or qualify returned column names. The constant provides a global value of the companion type, for ease of calling and avoidance of allocations.",
	UsageExample:        "pan generate ./pkg1 ./pkg2/pkg3",
	ArgsAccepted:        true,
	Flags: []clif.FlagDef{
		{
			Name:        "--type-suffix",
			Description: "the suffix to append to the struct type's name to construct the companion type's name",
			Default: []clif.FlagValue{
				{
					HasValue:     true,
					Raw:          "Columns",
					CanonicalKey: "--type-suffix",
				},
			},
		},
		{
			Name:        "--column-mismatch-check",
			Description: "whether to include a check in the init function of the generated file that ensures the mappings were generated from the same version of the code that's running",
			Default: []clif.FlagValue{
				{
					HasValue:     true,
					Raw:          "true",
					CanonicalKey: "--column-mismatch-check",
				},
			},
		},
		{
			Name:          "--all-columns-method",
			Description:   "the name of the method that should be generated to return all of the column names as a slice of strings. Can be specified multiple times. Will use the first name for each type that doesn't conflict with a field on the type, or omit the method if no non-conflicting name is specified. Set to the empty string to disable generating this method.",
			AllowMultiple: true,
			Default: []clif.FlagValue{
				{
					HasValue:     true,
					Raw:          "List",
					CanonicalKey: "--all-columns-method",
				},
				{
					HasValue:     true,
					Raw:          "All",
					CanonicalKey: "--all-columns-method",
				},
				{
					HasValue:     true,
					Raw:          "PanList",
					CanonicalKey: "--all-columns-method",
				},
				{
					HasValue:     true,
					Raw:          "PanAll",
					CanonicalKey: "--all-columns-method",
				},
			},
		},
		{
			Name:        "--generated-filename",
			Description: "the filename to save the generated code under in each package",
			Default: []clif.FlagValue{
				{
					HasValue:     true,
					Raw:          "pan_gen.go",
					CanonicalKey: "--generated-filename",
				},
			},
		},
	},
	Handler: generateHandlerBuilder{},
}

type generateHandlerBuilder struct{}

// Build creates a Handler by parsing the flags and args into the
// appropriate handler type.
func (generateHandlerBuilder) Build(ctx context.Context, flags clif.FlagSet, args []string, resp *clif.Response) clif.Handler { //nolint:ireturn // Have to return an interface to implement an interface
	var handler generateHandler
	err := flags.As(ctx, &handler)
	if err != nil {
		resp.Error.Write([]byte("Error parsing flags: " + err.Error())) //nolint:errcheck // nothing to be done if we can't report errors
		resp.Code = 1
		return nil
	}
	handler.Packages = args
	return handler
}

type generateHandler struct {
	TypeSuffix          string   `flag:"type-suffix"`
	ColumnMismatchCheck bool     `flag:"column-mismatch-check"`
	AllColumnsMethod    []string `flag:"all-columns-method"`
	GeneratedFilename   string   `flag:"generated-filename"`
	Packages            []string `flag:"-"`
}

// Handle is a method that will be called when the command is executed.
// It should contain the business logic of the command.
func (handler generateHandler) Handle(ctx context.Context, resp *clif.Response) {
	implPackages, err := FindSQLTableNamers(ctx, "", os.Args[1:])
	if err != nil {
		fmt.Fprintln(resp.Error, "Error searching for SQLTableNamer implementations:", err) //nolint:errcheck // nothing to be done if we can't report errors
		resp.Code = 1
		return
	}
	for dir, impls := range implPackages {
		handler.generateForPackage(ctx, dir, impls, resp)
		if resp.Code > 0 {
			return
		}
	}
}

func (handler generateHandler) generateForPackage(_ context.Context, dir string, impls []SQLTableNamerImpl, resp *clif.Response) {
	var packageName, packagePath string
	columnsByFieldByType := map[string]map[string]string{}
	for _, impl := range impls {
		if packageName == "" {
			packageName = impl.PackageName
		}
		if packagePath == "" {
			packagePath = impl.PackagePath
		}
		structInfo, ok := impl.Type.Underlying().(*types.Struct)
		if !ok {
			fmt.Fprintf(resp.Error, "%s in %s (package %s) is a %T, not a *types.Struct\r\n", impl.Name, impl.SourceDir, impl.PackagePath, impl.Type.Underlying()) //nolint:errcheck // nothing to be done if we can't report errors
			resp.Code = 1
			return
		}
		columnsByFieldByType[impl.Name] = map[string]string{}
		for fieldNum := range structInfo.NumFields() {
			field := structInfo.Field(fieldNum)
			tag := structInfo.Tag(fieldNum)
			columnName := pan.ColumnName(field.Name(), tag)
			if columnName == "" {
				continue
			}
			columnsByFieldByType[impl.Name][field.Name()] = columnName
		}
	}
	file := jen.NewFile(packageName)
	file.HeaderComment(`Code generated by darlinggo.co/pan. DO NOT EDIT.`)
	typeNames := slices.Collect(maps.Keys(columnsByFieldByType))
	slices.Sort(typeNames)
	file.Const().DefsFunc(func(g *jen.Group) {
		for _, name := range typeNames {
			g.Comment(fmt.Sprintf("%s provides helper functions to retrieve the mapped column names for fields on [%s].", name+handler.TypeSuffix, name))
			g.Id(name + handler.TypeSuffix).Id("columns_" + name).Op("=").Lit(0)
		}
	})
	if handler.ColumnMismatchCheck {
		file.Func().Id("init").Params().BlockFunc(func(initGroup *jen.Group) {
			for _, name := range typeNames {
				columns := slices.Collect(maps.Values(columnsByFieldByType[name]))
				slices.Sort(columns)
				initGroup.Id("sortedColumns_"+name).Op(":=").Qual("darlinggo.co/pan", "Columns").Call(jen.Id(name).Block())
				initGroup.Qual("slices", "Sort").Call(jen.Id("sortedColumns_" + name))
				initGroup.If(
					jen.Op("!").Qual("slices", "Equal").Call(
						jen.Id("sortedColumns_"+name),
						jen.Index().String().ValuesFunc(func(columnsGroup *jen.Group) {
							for _, column := range columns {
								columnsGroup.Line().Lit(column)
							}
							columnsGroup.Line().Empty()
						}),
					).Block(jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit(fmt.Sprintf("Column mismatch for %s.%s! This is usually caused by an out-of-date %s file. Try regenerating it.\r\n\r\nRuntime values: %%v\r\nGenerated values: %v", packagePath, name, handler.GeneratedFilename, pan.ColumnList(columns))), jen.Id("sortedColumns_"+name)))))
			}
		})
	}
	for _, name := range typeNames {
		file.Type().Id("columns_" + name).Int8()
		fields := columnsByFieldByType[name]
		fieldNames := slices.Collect(maps.Keys(fields))
		slices.Sort(fieldNames)
		for _, fieldName := range fieldNames {
			column := fields[fieldName]
			file.Comment(fmt.Sprintf("%s returns the database column name (%q) that [%s.%s] maps to.", fieldName, column, name, fieldName))
			file.Func().Params(jen.Id("columns_" + name)).Id(fieldName).Params(jen.Id("flags").Op("...").Qual("darlinggo.co/pan", "Flag")).String().Block(
				jen.Return(jen.Qual("darlinggo.co/pan", "DecorateColumn").Call(jen.Lit(column), jen.Qual("darlinggo.co/pan", "Table").Call(jen.Id(name).Block()), jen.Id("flags").Op("..."))),
			)
		}
		for _, methodName := range handler.AllColumnsMethod {
			if methodName == "" {
				continue
			}
			if slices.Contains(fieldNames, methodName) {
				continue
			}
			file.Comment(fmt.Sprintf("%s returns a sorted list of all database columns mapped to [%s].", methodName, name))
			file.Func().Params(jen.Id("columns_"+name)).Id(methodName).Params(jen.Id("flags").Op("...").Qual("darlinggo.co/pan", "Flag")).Index().String().Block(
				jen.Id("tableName").Op(":=").Qual("darlinggo.co/pan", "Table").Call(jen.Id(name).Block()),
				jen.Return(jen.Index().String().ValuesFunc(func(returnValues *jen.Group) {
					for _, fieldName := range fieldNames {
						column := fields[fieldName]
						returnValues.Line().Qual("darlinggo.co/pan", "DecorateColumn").Call(jen.Lit(column), jen.Id("tableName"), jen.Id("flags").Op("..."))
					}
					returnValues.Line().Empty()
				})),
			)
			break
		}
	}
	outputPath := filepath.Join(dir, handler.GeneratedFilename)
	output, err := os.Create(outputPath) //nolint:gosec // the arguments come from a CLI; no new security issues are created
	if err != nil {
		fmt.Fprintf(resp.Error, "Error writing to %s: %s\r\n", outputPath, err.Error()) //nolint:errcheck // nothing to be done if we can't report errors
		resp.Code = 1
		return
	}
	defer func() {
		err = output.Close()
		if err != nil {
			fmt.Fprintf(resp.Error, "Error closing %s after writing: %s\r\n", outputPath, err.Error()) //nolint:errcheck // nothing to be done if we can't report errors
			resp.Code = 1
		}
	}()
	err = file.Render(output)
	if err != nil {
		fmt.Fprintf(resp.Error, "Error generating helper code for %s: %s\r\n", packagePath, err.Error()) //nolint:errcheck // nothing to be done if we can't report errors
		resp.Code = 1
		return
	}
	fmt.Fprintln(resp.Output, "Generated SQL column mapping code for", packagePath, "in", outputPath) //nolint:errcheck // nothing to be done if we can't report to stdout
}
