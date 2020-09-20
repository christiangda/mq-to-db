package consumer

import (
	"testing"
	"time"
)

func TestConfig_GetURI(t *testing.T) {
	type fields struct {
		Address  string
		Port     int
		Username string
		Password string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "EmptyUsernameValues",
			fields: fields{
				Username: "",
				Password: "pass",
				Address:  "127.0.0.1",
				Port:     1234,
			},
			want:    "amqp://127.0.0.1:1234/",
			wantErr: false,
		},
		{
			name: "EmptyPasswordValues",
			fields: fields{
				Username: "user",
				Password: "",
				Address:  "127.0.0.1",
				Port:     1234,
			},
			want:    "amqp://user:@127.0.0.1:1234/",
			wantErr: false,
		},
		{
			name: "EmptyUserAndPassValues",
			fields: fields{
				Username: "",
				Password: "",
				Address:  "127.0.0.1",
				Port:     1234,
			},
			want:    "amqp://127.0.0.1:1234/",
			wantErr: false,
		},
		{
			name: "AllValues",
			fields: fields{
				Username: "user",
				Password: "pass",
				Address:  "127.0.0.1",
				Port:     1234,
			},
			want:    "amqp://user:pass@127.0.0.1:1234/",
			wantErr: false,
		},
		{
			name: "EmptyPortValue",
			fields: fields{
				Username: "user",
				Password: "pass",
				Address:  "127.0.0.1",
				Port:     0,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "EmptyAddress",
			fields: fields{
				Username: "user",
				Password: "pass",
				Address:  "",
				Port:     1234,
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "AllEmpty",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &Config{
				Address:  tt.fields.Address,
				Port:     tt.fields.Port,
				Username: tt.fields.Username,
				Password: tt.fields.Password,
			}
			got, err := cc.GetURI()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GetURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.GetURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessages_Ack(t *testing.T) {
	type fields struct {
		ContentType     string
		ContentEncoding string
		MessageID       string
		Priority        Priority
		ConsumerTag     string
		Timestamp       time.Time
		Exchange        string
		RoutingKey      string
		Payload         []byte
		Acknowledger    Acknowledger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "NilAcknowledgerReturnError",
			fields: fields{
				Acknowledger: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Messages{
				ContentType:     tt.fields.ContentType,
				ContentEncoding: tt.fields.ContentEncoding,
				MessageID:       tt.fields.MessageID,
				Priority:        tt.fields.Priority,
				ConsumerTag:     tt.fields.ConsumerTag,
				Timestamp:       tt.fields.Timestamp,
				Exchange:        tt.fields.Exchange,
				RoutingKey:      tt.fields.RoutingKey,
				Payload:         tt.fields.Payload,
				Acknowledger:    tt.fields.Acknowledger,
			}
			if err := m.Ack(); (err != nil) != tt.wantErr {
				t.Errorf("Messages.Ack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessages_Reject(t *testing.T) {
	type fields struct {
		ContentType     string
		ContentEncoding string
		MessageID       string
		Priority        Priority
		ConsumerTag     string
		Timestamp       time.Time
		Exchange        string
		RoutingKey      string
		Payload         []byte
		Acknowledger    Acknowledger
	}
	type args struct {
		requeue bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "NilAcknowledgerReturnError",
			fields: fields{
				Acknowledger: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Messages{
				ContentType:     tt.fields.ContentType,
				ContentEncoding: tt.fields.ContentEncoding,
				MessageID:       tt.fields.MessageID,
				Priority:        tt.fields.Priority,
				ConsumerTag:     tt.fields.ConsumerTag,
				Timestamp:       tt.fields.Timestamp,
				Exchange:        tt.fields.Exchange,
				RoutingKey:      tt.fields.RoutingKey,
				Payload:         tt.fields.Payload,
				Acknowledger:    tt.fields.Acknowledger,
			}
			if err := m.Reject(tt.args.requeue); (err != nil) != tt.wantErr {
				t.Errorf("Messages.Reject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
