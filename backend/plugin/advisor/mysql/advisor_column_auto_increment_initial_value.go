package mysql

// Framework code is generated by the generator.

import (
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementInitialValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLAutoIncrementColumnInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLAutoIncrementColumnInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLAutoIncrementColumnInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
}

// ColumnAutoIncrementInitialValueAdvisor is the advisor checking for auto-increment column initial value.
type ColumnAutoIncrementInitialValueAdvisor struct {
}

// Check checks for auto-increment column initial value.
func (*ColumnAutoIncrementInitialValueAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &columnAutoIncrementInitialValueChecker{
		level: level,
		title: string(ctx.Rule.Type),
		value: payload.Number,
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type columnAutoIncrementInitialValueChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	value      int
}

func (checker *columnAutoIncrementInitialValueChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
		if option.AUTO_INCREMENT_SYMBOL() == nil || option.Ulonglong_number() == nil {
			continue
		}

		base := 10
		bitSize := 0
		value, err := strconv.ParseUint(option.Ulonglong_number().GetText(), base, bitSize)
		if err != nil {
			continue
		}
		if value != uint64(checker.value) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.AutoIncrementColumnInitialValueNotMatch,
				Title:   checker.title,
				Content: fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, checker.value),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *columnAutoIncrementInitialValueChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}

	// alter table option.
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		if option == nil {
			continue
		}
		for _, tableOption := range option.AllCreateTableOption() {
			if tableOption == nil || tableOption.AUTO_INCREMENT_SYMBOL() == nil || tableOption.Ulonglong_number() == nil {
				continue
			}

			base := 10
			bitSize := 0
			value, err := strconv.ParseUint(tableOption.Ulonglong_number().GetText(), base, bitSize)
			if err != nil {
				continue
			}
			if value != uint64(checker.value) {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.AutoIncrementColumnInitialValueNotMatch,
					Title:   checker.title,
					Content: fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, checker.value),
					Line:    checker.baseLine + ctx.GetStart().GetLine(),
				})
			}
		}
	}
}
