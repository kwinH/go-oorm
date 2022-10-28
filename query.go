package oorm

import (
	"database/sql"
	"encoding/json"
	"oorm/schema"
	"reflect"
)

func (d *DB) Raw(sql string, param ...interface{}) *DB {
	db := d.getInstance()
	db.sql = sql
	db.bindings = param
	return db
}

func (d *DB) WithDelete() *DB {
	db := d.getInstance()
	db.withDel = true
	return db
}

func (d *DB) Get(value interface{}) error {
	db := d.getInstance()

	tableInfo := db.getTableInfo(value)

	if db.sql == "" {
		if model, ok := tableInfo.Model.(IBeforeQuery); ok {
			err := model.BeforeQuery(db)
			if err != nil {
				return err
			}
		}

		if len(db.b.GetField()) == 0 {
			db.b.Select(db.getField()...)
		}

		if db.b.GetTable() == "" {
			db.b.Table(tableInfo.TableName)
		}

		if db.withDel == false {
			db.WhereNull(tableInfo.TableName + ".deleted_at")
		}

		db.sql, db.bindings = db.b.ToSql()
	}

	rows, err := db.Query(db.sql, db.bindings...)

	if err != nil {
		return err
	}

	withs := db.makeWiths(tableInfo)

	defer rows.Close()

	var dests []reflect.Value
	for rows.Next() {
		dest, err1 := db.rowHandle(tableInfo, rows)
		if err1 == nil {
			dests = append(dests, dest)
			db.getWiths(withs, dest)
		} else {
			db.AddError(err)
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	db.relationships(withs)

	db.setDestRelationships(dests, withs, tableInfo.Value)

	if db.Error != nil {
		return db.Error
	}

	return nil
}

func (d *DB) Find(value interface{}, id int64) error {
	db := d.getInstance()

	tableInfo := db.Parse(value)

	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()
	if err := db.Where(tableInfo.PrimaryKey.FieldName, id).Limit(1).Get(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return ErrNotFind
	}
	dest.Set(destSlice.Index(0))
	return nil
}

func (d *DB) First(value interface{}) error {
	db := d.getInstance()

	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()
	if err := db.Limit(1).Get(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return ErrNotFind
	}
	dest.Set(destSlice.Index(0))
	return nil
}

func (d *DB) rowHandle(tableInfo *schema.Schema, rows *sql.Rows) (dest reflect.Value, err error) {
	dest = reflect.New(tableInfo.Type).Elem()
	var values = make([]interface{}, len(tableInfo.Fields))
	var jsons = make(map[string]*[]byte)

	for i, field := range tableInfo.Fields {

		if field.IsJson {
			var str []byte
			jsons[field.Name] = &str
			values[i] = jsons[field.Name]
		} else {
			values[i] = dest.FieldByName(field.Name).Addr().Interface()
		}

	}
	if err = rows.Scan(values...); err != nil {
		return
	}

	for fieldName, data := range jsons {
		err = json.Unmarshal(*data, dest.FieldByName(fieldName).Addr().Interface())
		if err != nil {
			return
		}
	}

	if model, ok := dest.Addr().Interface().(IGetAttr); ok {
		model.GetAttr()
	}

	if model, ok := dest.Addr().Interface().(IAfterQuery); ok {
		if err = model.AfterQuery(d); err != nil {
			return
		}
	}
	return
}