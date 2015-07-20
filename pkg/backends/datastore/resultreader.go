package datastore

import (
	"database/sql/driver"
	"io"
	"time"

	//"golang.org/x/net/context"
	//"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"

	u "github.com/araddon/gou"
	"github.com/araddon/qlbridge/datasource"
	"github.com/araddon/qlbridge/expr"
	"github.com/araddon/qlbridge/value"
	"github.com/dataux/dataux/pkg/models"
)

var (
	_ models.ResultProvider = (*ResultReader)(nil)

	// Ensure we implement datasource.DataSource, Scanner
	_ datasource.DataSource = (*ResultReader)(nil)
	_ datasource.Scanner    = (*ResultReader)(nil)
)

// Google Datastore ResultReader implements result paging, reading
// - driver.Rows
type ResultReader struct {
	exit          <-chan bool
	finalized     bool
	hasprojection bool
	cursor        int
	proj          *expr.Projection
	cols          []string
	Vals          [][]driver.Value
	Total         int
	Req           *SqlToDatstore
}

// A wrapper, allowing us to implement sql/driver Next() interface
//   which is different than qlbridge/datasource Next()
type ResultReaderNext struct {
	*ResultReader
}

func NewResultReader(req *SqlToDatstore) *ResultReader {
	m := &ResultReader{}
	m.Req = req
	return m
}

func (m *ResultReader) Close() error { return nil }

func (m *ResultReader) buildProjection() {

	if m.hasprojection {
		return
	}
	m.hasprojection = true
	m.proj = expr.NewProjection()
	cols := m.proj.Columns
	sql := m.Req.sel
	if sql.Star {
		// Select Each field, grab fields from Table Schema
		for _, fld := range m.Req.tbl.Fields {
			cols = append(cols, expr.NewResultColumn(fld.Name, len(cols), nil, fld.Type))
		}
	} else if sql.CountStar() {
		// Count *
		cols = append(cols, expr.NewResultColumn("count", len(cols), nil, value.IntType))
	} else {
		for _, col := range m.Req.sel.Columns {
			if fld, ok := m.Req.tbl.FieldMap[col.SourceField]; ok {
				//u.Debugf("column: %#v", col)
				cols = append(cols, expr.NewResultColumn(col.SourceField, len(cols), col, fld.Type))
			} else {
				u.Debugf("Could not find: '%v' in %#v", col.SourceField, m.Req.tbl.FieldMap)
				u.Warnf("%#v", col)
			}
		}
	}
	colNames := make([]string, len(cols))
	for i, col := range cols {
		colNames[i] = col.As
	}
	m.cols = colNames
	m.proj.Columns = cols
	//u.Debugf("leaving Columns:  %v", len(m.proj.Columns))
}

func (m *ResultReader) Tables() []string {
	return nil
}

func (m *ResultReader) Columns() []string {
	return m.cols
}

func (m *ResultReader) Projection() (*expr.Projection, error) {
	m.buildProjection()
	return m.proj, nil
}

func (m *ResultReader) Open(connInfo string) (datasource.SourceConn, error) {
	panic("Not implemented")
	return m, nil
}

func (m *ResultReader) Schema() *models.Schema {
	return m.Req.tbl.Schema
}

func (m *ResultReader) MesgChan(filter expr.Node) <-chan datasource.Message {
	iter := m.CreateIterator(filter)
	return datasource.SourceIterChannel(iter, filter, m.exit)
}

func (m *ResultReader) CreateIterator(filter expr.Node) datasource.Iterator {
	return &ResultReaderNext{m}
}

// Finalize maps the Mongo Documents/results into
//    [][]interface{}   which is compabitble with sql/driver values
//
func (m *ResultReader) Finalize() error {

	m.finalized = true
	m.buildProjection()

	defer func() {
		u.Debugf("nice, finalize vals in ResultReader: %v", len(m.Vals))
	}()

	sql := m.Req.sel
	q := m.Req.query

	m.Vals = make([][]driver.Value, 0)

	if sql.CountStar() {
		// Count *
		vals := make([]driver.Value, 1)
		ct, err := q.Count(m.Req.dsCtx)
		if err != nil {
			u.Errorf("could not get count: %v", err)
			return err
		}
		vals[0] = ct
		m.Vals = append(m.Vals, vals)
		return nil
	}

	cols := m.proj.Columns
	if len(cols) == 0 {
		u.Errorf("WTF?  no cols? %v", cols)
	} else {
		//q = q.Project(m.cols...)
		//q.Project(m.cols...)
	}

	n := time.Now()
	iter := q.Run(m.Req.dsCtx)
	for {
		row := Row{}
		key, err := iter.Next(&row)
		if err != nil {
			if err == datastore.Done {
				break
			}
			u.Errorf("error: %v", err)
			break
		}
		row.key = key
		m.Vals = append(m.Vals, row.Vals)
		u.Debugf("vals:  %v", row.Vals)
	}
	u.Infof("finished query, took: %v for %v rows", time.Now().Sub(n), len(m.Vals))
	return nil
}

// Implement sql/driver Rows Next() interface
func (m *ResultReader) Next(row []driver.Value) error {
	if m.cursor >= len(m.Vals) {
		return io.EOF
	}
	m.cursor++
	u.Debugf("ResultReader.Next():  cursor:%v  %v", m.cursor, len(m.Vals[m.cursor-1]))
	for i, val := range m.Vals[m.cursor-1] {
		row[i] = val
	}
	return nil
}

func (m *ResultReaderNext) Next() datasource.Message {
	select {
	case <-m.exit:
		return nil
	default:
		if !m.finalized {
			if err := m.Finalize(); err != nil {
				u.Errorf("Could not finalize: %v", err)
				return nil
			}
		}
		if m.cursor >= len(m.Vals) {
			return nil
		}
		m.cursor++
		//u.Debugf("ResultReader.Next():  cursor:%v  %v", m.cursor, len(m.Vals[m.cursor-1]))
		return &datasource.SqlDriverMessage{m.Vals[m.cursor-1], uint64(m.cursor)}
	}
}

func pageRowsQuery(iter *datastore.Iterator) []Row {
	rows := make([]Row, 0)
	for {
		row := Row{}
		if key, err := iter.Next(&row); err != nil {
			if err == datastore.Done {
				break
			}
			u.Errorf("error: %v", err)
			break
		} else {
			row.key = key
			//u.Debugf("key:  %#v", key)
			rows = append(rows, row)
		}
	}
	return rows
}

type Row struct {
	Vals  []driver.Value
	props []datastore.Property
	key   *datastore.Key
}

func (m *Row) Load(props []datastore.Property) error {
	m.Vals = make([]driver.Value, len(props))
	m.props = props
	//u.Infof("Load: %#v", props)
	for i, p := range props {
		//u.Infof("prop: %#v", p)
		m.Vals[i] = p.Value
	}
	return nil
}
func (m *Row) Save() ([]datastore.Property, error) {
	return nil, nil
}
