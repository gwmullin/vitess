// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlparser

import "github.com/youtube/vitess/go/sqltypes"

// SQLNode defines the interface for all nodes
// generated by the parser.
type SQLNode interface {
	Format(buf *TrackedBuffer)
}

// String returns a string representation of an SQLNode.
func String(node SQLNode) string {
	buf := NewTrackedBuffer(nil)
	buf.Fprintf("%v", node)
	return buf.String()
}

// Statement represents a statement. It can be Select,
// Union, Insert, Update, Delete, Set, DDLSimple, Rename.
type Statement interface {
	statement()
	SQLNode
}

// SelectStatement any SELECT statement. It can be Select, Union.
type SelectStatement interface {
	selectStatement()
	statement()
	SQLNode
}

// Select represents a SELECT statement.
// Lock can be "", " for update", " lock in share mode".
type Select struct {
	Comments    Comments
	Distinct    Distinct
	SelectExprs SelectExprs
	From        TableExprs
	Where       *Where
	GroupBy     GroupBy
	Having      *Where
	OrderBy     OrderBy
	Limit       *Limit
	Lock        string
}

func (*Select) statement() {}

func (*Select) selectStatement() {}

func (node *Select) Format(buf *TrackedBuffer) {
	buf.Fprintf("select %v%v%v from %v%v%v%v%v%v%s",
		node.Comments, node.Distinct, node.SelectExprs,
		node.From, node.Where,
		node.GroupBy, node.Having, node.OrderBy,
		node.Limit, node.Lock)
}

// Union represents a UNION statement.
// Type can be "union", "union all", "minus", "except",
// "intersect".
type Union struct {
	Type             string
	Select1, Select2 SelectStatement
}

func (*Union) statement() {}

func (*Union) selectStatement() {}

func (node *Union) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v", node.Select1, node.Type, node.Select2)
}

// Insert represents an INSERT statement.
// Rows can be *Subselect, Values.
type Insert struct {
	Comments Comments
	Table    *TableName
	Columns  Columns
	Rows     SQLNode
	OnDup    UpdateExprs
}

func (*Insert) statement() {}

func (node *Insert) Format(buf *TrackedBuffer) {
	buf.Fprintf("insert %vinto %v%v %v",
		node.Comments,
		node.Table, node.Columns, node.Rows, node.OnDup)
	if node.OnDup != nil {
		buf.Fprintf(" on duplicate key update %v", node.OnDup)
	}
}

// Update represents an UPDATE statement.
type Update struct {
	Comments Comments
	Table    *TableName
	List     UpdateExprs
	Where    *Where
	OrderBy  OrderBy
	Limit    *Limit
}

func (*Update) statement() {}

func (node *Update) Format(buf *TrackedBuffer) {
	buf.Fprintf("update %v%v set %v%v%v%v",
		node.Comments, node.Table,
		node.List, node.Where, node.OrderBy, node.Limit)
}

// Delete represents a DELETE statement.
type Delete struct {
	Comments Comments
	Table    *TableName
	Where    *Where
	OrderBy  OrderBy
	Limit    *Limit
}

func (*Delete) statement() {}

func (node *Delete) Format(buf *TrackedBuffer) {
	buf.Fprintf("delete %vfrom %v%v%v%v",
		node.Comments,
		node.Table, node.Where, node.OrderBy, node.Limit)
}

// Set represents a SET statement.
type Set struct {
	Comments Comments
	Updates  UpdateExprs
}

func (*Set) statement() {}

func (node *Set) Format(buf *TrackedBuffer) {
	buf.Fprintf("set %v%v", node.Comments, node.Updates)
}

// DDLSimple represents a CREATE, ALTER or DROP statement.
type DDLSimple struct {
	Action int
	Table  []byte
}

func (*DDLSimple) statement() {}

func (node *DDLSimple) Format(buf *TrackedBuffer) {
	switch node.Action {
	case CREATE:
		buf.Fprintf("create table %s", node.Table)
	case ALTER:
		buf.Fprintf("alter table %s", node.Table)
	case DROP:
		buf.Fprintf("drop table %s", node.Table)
	default:
		panic("unreachable")
	}
}

// Rename represents a RENAME statement.
type Rename struct {
	OldName, NewName []byte
}

func (*Rename) statement() {}

func (node *Rename) Format(buf *TrackedBuffer) {
	buf.Fprintf("rename table %s %s", node.OldName, node.NewName)
}

// Comments represents a list of comments.
type Comments []Comment

func (node Comments) Format(buf *TrackedBuffer) {
	for _, c := range node {
		c.Format(buf)
	}
}

// Comment represents one comment.
type Comment []byte

func (comment Comment) Format(buf *TrackedBuffer) {
	buf.Fprintf("%s ", []byte(comment))
}

// Distinct specifies if DISTINCT was used.
type Distinct bool

func (node Distinct) Format(buf *TrackedBuffer) {
	if node {
		buf.Fprintf("distinct ")
	}
}

// SelectExprs represents SELECT expressions.
type SelectExprs []SelectExpr

func (node SelectExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// SelectExpr represents a SELECT expression.
// It can be StarExpr, NonStarExpr.
type SelectExpr interface {
	selectExpr()
	SQLNode
}

// StarExpr defines a '*' or 'table.*' expression.
type StarExpr struct {
	TableName []byte
}

func (*StarExpr) selectExpr() {}

func (node *StarExpr) Format(buf *TrackedBuffer) {
	if node.TableName != nil {
		buf.Fprintf("%s.", node.TableName)
	}
	buf.Fprintf("*")
}

// NonStarExpr defines a non-'*' select expr.
type NonStarExpr struct {
	Expr Expr
	As   []byte
}

func (*NonStarExpr) selectExpr() {}

func (node *NonStarExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v", node.Expr)
	if node.As != nil {
		buf.Fprintf(" as %s", node.As)
	}
}

// Columns represents an insert column list.
// The syntax for Columns is a subset of SelectExprs.
// So, it's castable to a SelectExprs and can be analyzed
// as such.
type Columns []SelectExpr

func (node Columns) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf("(%v)", SelectExprs(node))
}

// TableExprs represents a list of table expressions.
type TableExprs []TableExpr

func (node TableExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// TableExpr represents a table expression.
// It can be AliasedTableExpr, ParenTableExpr, JoinTableExpr.
type TableExpr interface {
	tableExpr()
	SQLNode
}

// AliasedTableExpr represents a table expression
// coupled with an optional alias or index hint.
type AliasedTableExpr struct {
	Expr  SQLNode
	As    []byte
	Hints *IndexHints
}

func (*AliasedTableExpr) tableExpr() {}

func (node *AliasedTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v", node.Expr)
	if node.As != nil {
		buf.Fprintf(" as %s", node.As)
	}
	if node.Hints != nil {
		// Hint node provides the space padding.
		buf.Fprintf("%v", node.Hints)
	}
}

// TableName represents a table  name.
// TODO(sougou): This is currently identical to ColName. Resolve.
type TableName struct {
	Name, Qualifier []byte
}

func (node *TableName) Format(buf *TrackedBuffer) {
	if node.Qualifier != nil {
		escape(buf, node.Qualifier)
		buf.Fprintf(".")
	}
	escape(buf, node.Name)
}

// ParenTableExpr represents a parenthesized TableExpr.
type ParenTableExpr struct {
	Expr TableExpr
}

func (*ParenTableExpr) tableExpr() {}

func (node *ParenTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Expr)
}

// JoinTableExpr represents a TableExpr that's a JOIN
// operation. Join can be "join", "straight_join", "left join",
// "right join", "cross join", natural join".
type JoinTableExpr struct {
	LeftExpr  TableExpr
	Join      string
	RightExpr TableExpr
	On        BoolExpr
}

func (*JoinTableExpr) tableExpr() {}

func (node *JoinTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v", node.LeftExpr, node.Join, node.RightExpr)
	if node.On != nil {
		buf.Fprintf(" on %v", node.On)
	}
}

// IndexHints represents a list of index hints.
// Type can be "use", "ignore" or "force".
// TODO(sougou): See if Indexes can reuse Columns.
type IndexHints struct {
	Type    string
	Indexes [][]byte
}

func (node *IndexHints) Format(buf *TrackedBuffer) {
	buf.Fprintf(" %s index ", node.Type)
	prefix := "("
	for _, n := range node.Indexes {
		buf.Fprintf("%s%s", prefix, n)
		prefix = ", "
	}
	buf.Fprintf(")")
}

// Where represents a WHERE or HAVING clause.
// Type can be "where", "having"
type Where struct {
	Type string
	Expr BoolExpr
}

func (node *Where) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" %s %v", node.Type, node.Expr)
}

// Expr represents an expression. It can be BoolExpr, ValExpr.
type Expr interface {
	expr()
	SQLNode
}

// BoolExpr represents a boolean expression.
// It can be AndExpr, OrExpr, NotExpr, ParenBoolExpr,
// ComparisonExpr, RangeCond, NullCheck, ExistsExpr
type BoolExpr interface {
	boolExpr()
	Expr
}

// AndExpr represents an AND expression.
type AndExpr struct {
	Left, Right BoolExpr
}

func (*AndExpr) expr()     {}
func (*AndExpr) boolExpr() {}

func (node *AndExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v and %v", node.Left, node.Right)
}

// OrExpr represents an OR expression.
type OrExpr struct {
	Left, Right BoolExpr
}

func (*OrExpr) expr()     {}
func (*OrExpr) boolExpr() {}

func (node *OrExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v or %v", node.Left, node.Right)
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr BoolExpr
}

func (*NotExpr) expr()     {}
func (*NotExpr) boolExpr() {}

func (node *NotExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("not %v", node.Expr)
}

// ParenBoolExpr represents a parenthesized boolean expression.
type ParenBoolExpr struct {
	Expr BoolExpr
}

func (*ParenBoolExpr) expr()     {}
func (*ParenBoolExpr) boolExpr() {}

func (node *ParenBoolExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Expr)
}

// ComparisonExpr represents a two-value comparison expression.
// Operator can be "=", ",", ">", "<=", ">=", "<>", "!=", "<=>",
// "in", "not in", "like", "not like".
type ComparisonExpr struct {
	Operator    string
	Left, Right ValExpr
}

func (*ComparisonExpr) expr()     {}
func (*ComparisonExpr) boolExpr() {}

func (node *ComparisonExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v", node.Left, node.Operator, node.Right)
}

// RangeCond represents a BETWEEN or a NOT BETWEEN expression.
// Operator can be "between", "not between".
type RangeCond struct {
	Operator string
	Left     ValExpr
	From, To ValExpr
}

func (*RangeCond) expr()     {}
func (*RangeCond) boolExpr() {}

func (node *RangeCond) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v and %v", node.Left, node.Operator, node.From, node.To)
}

// NullCheck represents an IS NULL or an IS NOT NULL expression.
// Operator can be "is null", "is not null".
type NullCheck struct {
	Operator string
	Expr     ValExpr
}

func (*NullCheck) expr()     {}
func (*NullCheck) boolExpr() {}

func (node *NullCheck) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s", node.Expr, node.Operator)
}

// ExistsExpr represents an EXISTS expression.
type ExistsExpr struct {
	Subquery *Subquery
}

func (*ExistsExpr) expr()     {}
func (*ExistsExpr) boolExpr() {}

func (node *ExistsExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("exists %v", node.Subquery)
}

// ValExpr represents a value expression.
type ValExpr interface {
	valExpr()
	Expr
}

// StringValue represents a string value.
type StringValue []byte

func (StringValue) expr()    {}
func (StringValue) valExpr() {}

func (node StringValue) Format(buf *TrackedBuffer) {
	s := sqltypes.MakeString([]byte(node))
	s.EncodeSql(buf)
}

// NumValue represents a number.
type NumValue []byte

func (NumValue) expr()    {}
func (NumValue) valExpr() {}

func (node NumValue) Format(buf *TrackedBuffer) {
	buf.Fprintf("%s", []byte(node))
}

// ValueArg represents a named bind var argument.
type ValueArg []byte

func (ValueArg) expr()    {}
func (ValueArg) valExpr() {}

func (node ValueArg) Format(buf *TrackedBuffer) {
	buf.WriteArg(string(node[1:]))
}

// NullValue represents a NULL value.
type NullValue struct{}

func (*NullValue) expr()    {}
func (*NullValue) valExpr() {}

func (node *NullValue) Format(buf *TrackedBuffer) {
	buf.Fprintf("null")
}

// ColName represents a column name.
type ColName struct {
	Name, Qualifier []byte
}

func (*ColName) expr()    {}
func (*ColName) valExpr() {}

func (node *ColName) Format(buf *TrackedBuffer) {
	if node.Qualifier != nil {
		escape(buf, node.Qualifier)
		buf.Fprintf(".")
	}
	escape(buf, node.Name)
}

func escape(buf *TrackedBuffer, name []byte) {
	if _, ok := keywords[string(name)]; ok {
		buf.Fprintf("`%s`", name)
	} else {
		buf.Fprintf("%s", name)
	}
}

// Tuple represents a tuple. It can be ValueTuple, Subquery.
type Tuple interface {
	tuple()
	ValExpr
}

// ValueTuple represents a tuple of actual values.
type ValueTuple ValExprs

func (ValueTuple) tuple()   {}
func (ValueTuple) expr()    {}
func (ValueTuple) valExpr() {}

func (node ValueTuple) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", ValExprs(node))
}

// ValExprs represents a list of value expressions.
// It's not a valid expression because it's not parenthesized.
type ValExprs []ValExpr

func (node ValExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// Subquery represents a subquery.
type Subquery struct {
	Select SelectStatement
}

func (*Subquery) tuple()   {}
func (*Subquery) expr()    {}
func (*Subquery) valExpr() {}

func (node *Subquery) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Select)
}

// BinaryExpr represents a binary value expression.
// Operator can be &, |, ^, +, -, *, /, %.
type BinaryExpr struct {
	Operator    byte
	Left, Right Expr
}

func (*BinaryExpr) expr()    {}
func (*BinaryExpr) valExpr() {}

func (node *BinaryExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v%c%v", node.Left, node.Operator, node.Right)
}

// UnaryExpr represents a unary value expression.
// Operator can be +, -, ~.
type UnaryExpr struct {
	Operator byte
	Expr     Expr
}

func (*UnaryExpr) expr()    {}
func (*UnaryExpr) valExpr() {}

func (node *UnaryExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%c%v", node.Operator, node.Expr)
}

// FuncExpr represents a function call.
type FuncExpr struct {
	Name     []byte
	Distinct bool
	Exprs    SelectExprs
}

func (*FuncExpr) expr()    {}
func (*FuncExpr) valExpr() {}

func (node *FuncExpr) Format(buf *TrackedBuffer) {
	var distinct string
	if node.Distinct {
		distinct = "distinct "
	}
	buf.Fprintf("%s(%s%v)", node.Name, distinct, node.Exprs)
}

// CaseExpr represents a CASE expression.
type CaseExpr struct {
	Expr  ValExpr
	Whens []*When
	Else  ValExpr
}

func (*CaseExpr) expr()    {}
func (*CaseExpr) valExpr() {}

func (node *CaseExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("case ")
	if node.Expr != nil {
		buf.Fprintf("%v ", node.Expr)
	}
	for _, when := range node.Whens {
		buf.Fprintf("%v ", when)
	}
	if node.Else != nil {
		buf.Fprintf("else %v ", node.Else)
	}
	buf.Fprintf("end")
}

// When represents a WHEN sub-expression.
type When struct {
	Cond BoolExpr
	Val  ValExpr
}

func (node *When) Format(buf *TrackedBuffer) {
	buf.Fprintf("when %v then %v", node.Cond, node.Val)
}

// Values represents a VALUES clause.
type Values []Tuple

func (node Values) Format(buf *TrackedBuffer) {
	prefix := "values "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// GroupBy represents a GROUP BY clause.
type GroupBy []ValExpr

func (node GroupBy) Format(buf *TrackedBuffer) {
	prefix := " group by "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// OrderBy represents an ORDER By clause.
type OrderBy []*Order

func (node OrderBy) Format(buf *TrackedBuffer) {
	prefix := " order by "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// Order represents an ordering expression.
// Direction can be "asc", "desc".
type Order struct {
	Expr      ValExpr
	Direction string
}

func (node *Order) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s", node.Expr, node.Direction)
}

// Limit represents a LIMIT clause.
type Limit struct {
	Offset, Rowcount ValExpr
}

func (node *Limit) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" limit ")
	if node.Offset != nil {
		buf.Fprintf("%v, ", node.Offset)
	}
	buf.Fprintf("%v", node.Rowcount)
}

// UpdateExprs represents a list of update expressions.
type UpdateExprs []*UpdateExpr

func (node UpdateExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// UpdateExpr represents an update expression.
type UpdateExpr struct {
	Name *ColName
	Expr ValExpr
}

func (node *UpdateExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v = %v", node.Name, node.Expr)
}
