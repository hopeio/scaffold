package excel

import (
	"database/sql"

	"github.com/xuri/excelize/v2"
)

// TODO
type NullString struct {
	sql.Null[string]
}

// String returns the underlying string value, ignoring the null flag.
func (n NullString) String() string {
	return n.V
}

// Export writes sql.Rows to an Excel file at the given filename, using column names as the header row.
func Export(rows *sql.Rows, filename string) error {
	f := excelize.NewFile()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	/*	types,err:=rows.ColumnTypes()
		if err != nil {
			return err
		}*/
	err = f.SetSheetRow("Sheet1", "A1", &columns)
	if err != nil {
		return err
	}
	columnValues := make([]NullString, len(columns))
	columnValuePtrs := make([]any, len(columns))
	for i := range columnValues {
		columnValuePtrs[i] = &columnValues[i]
	}
	row := 2
	for rows.Next() {
		err = rows.Scan(columnValuePtrs...)
		if err != nil {
			return err
		}
		cell, err := excelize.CoordinatesToCellName(1, row)
		if err != nil {
			return err
		}
		err = f.SetSheetRow("Sheet1", cell, &columnValues)
		if err != nil {
			return err
		}
		row++
	}
	err = rows.Close()
	if err != nil {
		return err
	}
	return f.SaveAs(filename)
}
