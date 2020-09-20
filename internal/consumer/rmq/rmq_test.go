package rmq

import (
	"reflect"
	"testing"

	"github.com/christiangda/mq-to-db/internal/consumer"
)

func TestNew(t *testing.T) {
	type args struct {
		c *consumer.Config
	}
	tests := []struct {
		name    string
		args    args
		want    *RMQ
		wantErr bool
	}{
		{
			name:    "EmptyConfGenerateError",
			args:    args{&consumer.Config{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NecessaryFieldsNoError",
			args: args{&consumer.Config{
				Address: "127.0.0.1",
				Port:    1572,
			}},
			want:    &RMQ{uri: "amqp://127.0.0.1:1572/"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.c)
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
