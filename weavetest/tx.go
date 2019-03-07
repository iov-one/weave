package weavetest

import "github.com/iov-one/weave"

type Tx struct {
	Msg weave.Msg
	Err error
}

var _ weave.Tx = (*Tx)(nil)

func (tx *Tx) GetMsg() (weave.Msg, error) {
	return tx.Msg, tx.Err
}

func (tx *Tx) Unmarshal([]byte) error {
	panic("not implemented")
}

func (tx *Tx) Marshal() ([]byte, error) {
	panic("not implemented")
}

type Msg struct {
	RoutePath  string
	Serialized []byte
	Err        error
}

var _ weave.Msg = (*Msg)(nil)

func (m *Msg) Path() string {
	return m.RoutePath
}

func (m *Msg) Unmarshal(b []byte) error {
	m.Serialized = b
	return m.Err
}

func (m *Msg) Marshal() ([]byte, error) {
	return m.Serialized, m.Err
}
