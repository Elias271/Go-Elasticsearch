package models

import (
	u "github.com/araddon/gou"

	"github.com/araddon/qlbridge/schema"
	"github.com/araddon/qlbridge/value"

	"github.com/dataux/dataux/vendored/mixer/mysql"
)

var (
	_ = u.EMPTY
)

/*
func DescribeTable(tbl *datasource.Table) (*membtree.StaticDataSource, *expr.Projection) {
	if len(tbl.Fields) == 0 {
		u.Warnf("NO Fields!!!!! for %s p=%p", tbl.Name, tbl)
	}
	proj := expr.NewProjection()
	for _, f := range datasource.DescribeHeaders {
		proj.AddColumnShort(string(f.Name), f.Type)
		//u.Debugf("found field:  vals=%#v", f)
	}
	tableVals := membtree.NewStaticDataSource("describetable", 0, tbl.DescribeValues, datasource.DescribeCols)
	return tableVals, proj
}

func ShowTables(s *datasource.Schema) (*membtree.StaticDataSource, *expr.Projection) {

	tables := s.Tables()
	vals := make([][]driver.Value, len(tables))
	idx := 0
	if len(tables) == 0 {
		u.Warnf("NO TABLES!!!!! for %+v", s)
	}
	for _, tbl := range tables {
		vals[idx] = []driver.Value{tbl}
		//u.Infof("found table: %v   vals=%v", tbl, vals[idx])
		idx++
	}
	showTableVals := membtree.NewStaticDataSource("schematables", 0, vals, []string{"Table"})
	proj := expr.NewProjection()
	proj.AddColumnShort("Table", value.StringType)
	//u.Infof("showtables:  %v", m.showTableVals)
	return showTableVals, proj
}

func ShowVariables(s *datasource.Schema, name string, val driver.Value) (*membtree.StaticDataSource, *expr.Projection) {
	/ *
	   MariaDB [(none)]> SHOW SESSION VARIABLES LIKE 'lower_case_table_names';
	   +------------------------+-------+
	   | Variable_name          | Value |
	   +------------------------+-------+
	   | lower_case_table_names | 0     |
	   +------------------------+-------+
	* /
	vals := make([][]driver.Value, 1)
	vals[0] = []driver.Value{name, val}
	dataSource := membtree.NewStaticDataSource("schematables", 0, vals, []string{"Variable_name", "Value"})
	p := expr.NewProjection()
	p.AddColumnShort("Variable_name", value.StringType)
	p.AddColumnShort("Value", value.StringType)
	return dataSource, p
}
*/

func TableToMysqlResultset(tbl *schema.Table) *mysql.Resultset {
	rs := new(mysql.Resultset)
	rs.Fields = mysql.DescribeHeaders
	rs.FieldNames = mysql.DescribeFieldNames
	for _, val := range tbl.DescribeValues {
		rs.AddRowValues(val)
	}
	return rs
}

/*
func (m *Table) AddMySqlField(fld *mysql.Field) {
	m.FieldsMySql = append(m.FieldsMySql, fld)
	if fld.FieldName == "" {
		fld.FieldName = string(fld.Name)
	}
	m.FieldMapMySql[fld.FieldName] = fld
}
*/

func FieldToMysql(f *schema.Field, s *schema.SourceSchema) *mysql.Field {
	switch f.Type {
	case value.StringType:
		return mysql.NewField(f.Name, s.Name, s.Name, f.Length, mysql.MYSQL_TYPE_STRING)
	case value.BoolType:
		return mysql.NewField(f.Name, s.Name, s.Name, 1, mysql.MYSQL_TYPE_TINY)
	case value.IntType:
		return mysql.NewField(f.Name, s.Name, s.Name, 32, mysql.MYSQL_TYPE_LONG)
	case value.NumberType:
		return mysql.NewField(f.Name, s.Name, s.Name, 64, mysql.MYSQL_TYPE_FLOAT)
	case value.TimeType:
		return mysql.NewField(f.Name, s.Name, s.Name, 8, mysql.MYSQL_TYPE_DATETIME)
	default:
		u.Warnf("Could not find mysql type for :%T", f.Type)
	}

	return nil
}
