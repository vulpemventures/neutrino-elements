package node

import (
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"testing"
)

func TestCheckHeadersInSequence(t *testing.T) {
	type args struct {
		headers protocol.MsgHeaders
		tip     *block.Header
	}
	var tests = []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1",
			args: args{
				headers: protocol.MsgHeaders{
					Headers: []*block.Header{
						{
							Height: 1,
						},
						{
							Height: 2,
						},
						{
							Height: 3,
						},
						{
							Height: 4,
						},
						{
							Height: 5,
						},
						{
							Height: 6,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "2",
			args: args{
				headers: protocol.MsgHeaders{
					Headers: []*block.Header{
						{
							Height: 1,
						},
						{
							Height: 2,
						},
						{
							Height: 3,
						},
						{
							Height: 29,
						},
						{
							Height: 5,
						},
						{
							Height: 6,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "3",
			args: args{
				headers: protocol.MsgHeaders{
					Headers: []*block.Header{
						{
							Height: 10,
						},
						{
							Height: 2,
						},
						{
							Height: 3,
						},
						{
							Height: 29,
						},
						{
							Height: 5,
						},
						{
							Height: 6,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "4",
			args: args{
				headers: protocol.MsgHeaders{
					Headers: []*block.Header{
						{
							Height: 1,
						},
						{
							Height: 2,
						},
						{
							Height: 3,
						},
						{
							Height: 29,
						},
						{
							Height: 5,
						},
						{
							Height: 7,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, checkHeadersInSequence(tt.args.headers), "checkHeadersInSequence(%v, %v)", tt.args.headers, tt.args.tip)
		})
	}
}
