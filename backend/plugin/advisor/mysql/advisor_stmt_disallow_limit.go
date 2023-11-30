package mysql

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
}

// DisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
type DisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*DisallowLimitAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &disallowLimitChecker{
		level: level,
		title: string(ctx.Rule.Type),
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

type disallowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	isInsertStmt bool
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	text         string
	line         int
}

func (checker *disallowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *disallowLimitChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.SimpleLimitClause() != nil && ctx.SimpleLimitClause().LIMIT_SYMBOL() != nil {
		checker.handleLimitClause(advisor.DeleteUseLimit, ctx.GetStart().GetLine())
	}
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *disallowLimitChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.SimpleLimitClause() != nil && ctx.SimpleLimitClause().LIMIT_SYMBOL() != nil {
		checker.handleLimitClause(advisor.UpdateUseLimit, ctx.GetStart().GetLine())
	}
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *disallowLimitChecker) EnterInsertStatement(_ *mysql.InsertStatementContext) {
	checker.isInsertStmt = true
}

// ExitInsertStatement is called when production insertStatement is exited.
func (checker *disallowLimitChecker) ExitInsertStatement(_ *mysql.InsertStatementContext) {
	checker.isInsertStmt = false
}

// EnterQueryExpression is called when production queryExpression is entered.
func (checker *disallowLimitChecker) EnterQueryExpression(ctx *mysql.QueryExpressionContext) {
	if !checker.isInsertStmt {
		return
	}
	if ctx.LimitClause() != nil && ctx.LimitClause().LIMIT_SYMBOL() != nil {
		checker.handleLimitClause(advisor.InsertUseLimit, ctx.GetStart().GetLine())
	}
}

func (checker *disallowLimitChecker) handleLimitClause(code advisor.Code, lineNumber int) {
	checker.adviceList = append(checker.adviceList, advisor.Advice{
		Status:  checker.level,
		Code:    code,
		Title:   checker.title,
		Content: fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", checker.text),
		Line:    checker.line + lineNumber,
	})
}
