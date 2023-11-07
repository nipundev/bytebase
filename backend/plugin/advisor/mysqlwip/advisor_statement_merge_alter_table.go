package mysqlwip

// Framework code is generated by the generator.

import (
	"fmt"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLMergeAlterTable, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for merging ALTER TABLE statements.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for merging ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementMergeAlterTableChecker{
		level:    level,
		title:    string(ctx.Rule.Type),
		tableMap: make(map[string]tableStatement),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.generateAdvice(), nil
}

type statementMergeAlterTableChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	text       string
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	tableMap   map[string]tableStatement
}

type tableStatement struct {
	name     string
	count    int
	lastLine int
}

func (checker *statementMergeAlterTableChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateTable is called when production createTable is entered.
func (checker *statementMergeAlterTableChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	checker.tableMap[tableName] = tableStatement{
		name:     tableName,
		count:    1,
		lastLine: checker.baseLine + ctx.GetStart().GetLine(),
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *statementMergeAlterTableChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, ok := checker.tableMap[tableName]
	if !ok {
		table = tableStatement{
			name:  tableName,
			count: 0,
		}
	}
	table.count++
	table.lastLine = checker.baseLine + ctx.GetStart().GetLine()
	checker.tableMap[tableName] = table
}

func (checker *statementMergeAlterTableChecker) generateAdvice() []advisor.Advice {
	var tableList []tableStatement
	for _, table := range checker.tableMap {
		tableList = append(tableList, table)
	}
	sort.Slice(tableList, func(i, j int) bool {
		return tableList[i].lastLine < tableList[j].lastLine
	})

	for _, table := range tableList {
		if table.count > 1 {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementRedundantAlterTable,
				Title:   checker.title,
				Content: fmt.Sprintf("There are %d statements to modify table `%s`", table.count, table.name),
				Line:    table.lastLine,
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList
}
