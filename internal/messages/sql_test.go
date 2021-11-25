package messages

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewSQL(t *testing.T) {
	type args struct {
		m []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SQL
		wantErr bool
	}{
		{
			name: "valid",
			args: args{m: []byte(`{"TYPE":"SQL","CONTENT":{"SERVER":"localhost","DB":"postgresql","USER":"postgres","PASS":"mysecretpassword","SENTENCE":"SELECT pg_sleep(1);"},"DATE":"2020-01-01 00:00:01.000000-1","APPID":"test","ADITIONAL":null,"ACK": false,"RESPONSE":null}`)},
			want: &SQL{
				Kind: "SQL",
				Content: struct {
					Server   string "json:\"SERVER\" yaml:\"SERVER\""
					DB       string "json:\"DB\" yaml:\"DB\""
					User     string "json:\"USER\" yaml:\"USER\""
					Pass     string "json:\"PASS\" yaml:\"PASS\""
					Sentence string "json:\"SENTENCE\" yaml:\"SENTENCE\""
				}{
					Server:   "localhost",
					DB:       "postgresql",
					User:     "postgres",
					Pass:     "mysecretpassword",
					Sentence: "SELECT pg_sleep(1);",
				},
				Date:  "2020-01-01 00:00:01.000000-1",
				AppID: "test",
				// Additional: "null",
				ACK: false,
				// Response:   "null",
			},
			wantErr: false,
		},
		{
			name:    "error", // the bool field ACK is string
			args:    args{m: []byte(`{"TYPE":"SQL","CONTENT":{"SERVER":"localhost","DB":"postgresql","USER":"postgres","PASS":"mysecretpassword","SENTENCE":"SELECT pg_sleep(1);"},"DATE":"2020-01-01 00:00:01.000000-1","APPID":"test","ADITIONAL":null,"ACK": "false","RESPONSE":null}`)},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSQL(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSQL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQL_ToJSON(t *testing.T) {
	type fields struct {
		Kind    string
		Content struct {
			Server   string
			DB       string
			User     string
			Pass     string
			Sentence string
		}
		Date       string
		AppID      string
		Additional string
		ACK        bool
		Response   string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Kind: "SQL",
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server:   "localhost",
					DB:       "postgresql",
					User:     "postgres",
					Pass:     "mysecretpassword",
					Sentence: "SELECT pg_sleep(1);",
				},
				Date:       "2020-01-01 00:00:01.000000-1",
				AppID:      "test",
				Additional: "null",
				ACK:        false,
				Response:   "null",
			},
			want: `{"TYPE":"SQL","CONTENT":{"SERVER":"localhost","DB":"postgresql","USER":"postgres","PASS":"mysecretpassword","SENTENCE":"SELECT pg_sleep(1);"},"DATE":"2020-01-01 00:00:01.000000-1","APPID":"test","ADITIONAL":"null","ACK":false,"RESPONSE":"null"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SQL{
				Kind: tt.fields.Kind,
				Content: struct {
					Server   string "json:\"SERVER\" yaml:\"SERVER\""
					DB       string "json:\"DB\" yaml:\"DB\""
					User     string "json:\"USER\" yaml:\"USER\""
					Pass     string "json:\"PASS\" yaml:\"PASS\""
					Sentence string "json:\"SENTENCE\" yaml:\"SENTENCE\""
				}{
					Server:   tt.fields.Content.Server,
					DB:       tt.fields.Content.DB,
					User:     tt.fields.Content.User,
					Pass:     tt.fields.Content.Pass,
					Sentence: tt.fields.Content.Sentence,
				},
				Date:       tt.fields.Date,
				AppID:      tt.fields.AppID,
				Additional: tt.fields.Additional,
				ACK:        tt.fields.ACK,
				Response:   tt.fields.Response,
			}
			if got := m.ToJSON(); got != tt.want {
				t.Errorf("SQL.ToJSON() = %v, want %v", got, tt.want)
			}
			if !bytes.Equal([]byte(m.ToJSON()), []byte(tt.want)) {
				t.Errorf("bytes of SQL.ToJSON() != bytes of %v", tt.want)
			}
		})
	}
}

func TestSQL_ToYAML(t *testing.T) {
	yamlFile := `TYPE: SQL
CONTENT:
    SERVER: localhost
    DB: postgresql
    USER: postgres
    PASS: mysecretpassword
    SENTENCE: SELECT pg_sleep(1);
DATE: 2020-01-01 00:00:01.000000-1
APPID: test
ADITIONAL: "null"
ACK: false
RESPONSE: "null"
`

	type fields struct {
		Kind    string
		Content struct {
			Server   string
			DB       string
			User     string
			Pass     string
			Sentence string
		}
		Date       string
		AppID      string
		Additional string
		ACK        bool
		Response   string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Kind: "SQL",
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server:   "localhost",
					DB:       "postgresql",
					User:     "postgres",
					Pass:     "mysecretpassword",
					Sentence: "SELECT pg_sleep(1);",
				},
				Date:       "2020-01-01 00:00:01.000000-1",
				AppID:      "test",
				Additional: "null",
				ACK:        false,
				Response:   "null",
			},
			want: yamlFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SQL{
				Kind: tt.fields.Kind,
				Content: struct {
					Server   string "json:\"SERVER\" yaml:\"SERVER\""
					DB       string "json:\"DB\" yaml:\"DB\""
					User     string "json:\"USER\" yaml:\"USER\""
					Pass     string "json:\"PASS\" yaml:\"PASS\""
					Sentence string "json:\"SENTENCE\" yaml:\"SENTENCE\""
				}{
					Server:   tt.fields.Content.Server,
					DB:       tt.fields.Content.DB,
					User:     tt.fields.Content.User,
					Pass:     tt.fields.Content.Pass,
					Sentence: tt.fields.Content.Sentence,
				},
				Date:       tt.fields.Date,
				AppID:      tt.fields.AppID,
				Additional: tt.fields.Additional,
				ACK:        tt.fields.ACK,
				Response:   tt.fields.Response,
			}
			if got := m.ToYAML(); got != tt.want {
				t.Errorf("SQL.ToYAML() = %v, want %v", got, tt.want)
			}
			if !bytes.Equal([]byte(m.ToYAML()), []byte(tt.want)) {
				t.Errorf("bytes of SQL.ToYAML() != bytes of %v", tt.want)
			}
		})
	}
}

func TestSQL_ValidDataConn(t *testing.T) {
	type fields struct {
		Content struct {
			Server   string
			DB       string
			User     string
			Pass     string
			Sentence string
		}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server:   "localhost",
					DB:       "postgresql",
					User:     "postgres",
					Pass:     "mysecretpassword",
					Sentence: "SELECT pg_sleep(1);",
				},
			},
			want: true,
		},
		{
			name: "invalid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server: "",
					DB:     "postgresql",
					User:   "postgres",
					Pass:   "mysecretpassword",
				},
			},
			want: false,
		},
		{
			name: "invalid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server: "localhost",
					DB:     "",
					User:   "postgres",
					Pass:   "mysecretpassword",
				},
			},
			want: false,
		},
		{
			name: "invalid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server: "localhost",
					DB:     "postgresql",
					User:   "",
					Pass:   "mysecretpassword",
				},
			},
			want: false,
		},
		{
			name: "invalid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server: "localhost",
					DB:     "postgresql",
					User:   "postgre",
					Pass:   "",
				},
			},
			want: false,
		},
		{
			name: "invalid",
			fields: fields{
				Content: struct {
					Server   string
					DB       string
					User     string
					Pass     string
					Sentence string
				}{
					Server: "",
					DB:     "",
					User:   "",
					Pass:   "",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SQL{
				Content: struct {
					Server   string "json:\"SERVER\" yaml:\"SERVER\""
					DB       string "json:\"DB\" yaml:\"DB\""
					User     string "json:\"USER\" yaml:\"USER\""
					Pass     string "json:\"PASS\" yaml:\"PASS\""
					Sentence string "json:\"SENTENCE\" yaml:\"SENTENCE\""
				}{
					Server:   tt.fields.Content.Server,
					DB:       tt.fields.Content.DB,
					User:     tt.fields.Content.User,
					Pass:     tt.fields.Content.Pass,
					Sentence: tt.fields.Content.Sentence,
				},
			}
			if got := m.ValidDataConn(); got != tt.want {
				t.Errorf("SQL.ValidDataConn() = %v, want %v", got, tt.want)
			}
		})
	}
}
