package queue

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		c *Config
	}
	tests := []struct {
		name    string
		args    args
		want    *RabbitMQ
		wantErr bool
	}{
		{
			name:    "EmptyConfGenerateError",
			args:    args{&Config{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NecessaryFieldsNoError",
			args: args{&Config{
				Address: "127.0.0.1",
				Port:    1572,
			}},
			want:    &RabbitMQ{uri: "amqp://127.0.0.1:1572/"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRabbitMQ(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
