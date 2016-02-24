package sftp

import (
	"io"
	"testing"
)

func clientServerPair(t *testing.T) (*Client, *Server) {
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	server, err := NewServer(serverRead, serverWrite)
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve()
	client, err := NewClientPipe(clientRead, clientWrite)
	if err != nil {
		t.Fatal(err)
	}
	return client, server
}

type sshFxpTestBadExtendedPacket struct {
	ID        uint32
	Extension string
	Data      string
}

func (p sshFxpTestBadExtendedPacket) id() uint32 { return p.ID }

func (p sshFxpTestBadExtendedPacket) MarshalBinary() ([]byte, error) {
	l := 1 + 4 + 4 + // type(byte) + uint32 + uint32
		len(p.Extension) +
		len(p.Data)

	b := make([]byte, 0, l)
	b = append(b, ssh_FXP_EXTENDED)
	b = marshalUint32(b, p.ID)
	b = marshalString(b, p.Extension)
	b = marshalString(b, p.Data)
	return b, nil
}

// test that errors are sent back when we request an invalid extended packet operation
func TestInvalidExtendedPacket(t *testing.T) {
	client, _ := clientServerPair(t)
	defer client.Close()
	badPacket := sshFxpTestBadExtendedPacket{client.nextID(), "thisDoesn'tExist", "foobar"}
	_, _, err := client.sendRequest(badPacket)
	if err != nil {
		t.Log(err)
	} else {
		t.Fatal("expected error from bad packet")
	}

	// try to stat a file; the client should have shut down.
	filePath := "/etc/passwd"
	_, err = client.Stat(filePath)
	if err != errClientRecvFinished {
		t.Fatal(err)
	}
}
