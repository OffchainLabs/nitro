package testhelpers

import (
	"net"
	"testing"
)

func TestFreeTCPPortListener(t *testing.T) {
	aListener, err := FreeTCPPortListener()
	if err != nil {
		t.Fatal(err)
	}
	bListener, err := FreeTCPPortListener()
	if err != nil {
		t.Fatal(err)
	}
	aTCPAddr, ok := aListener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("aListener.Addr() is not a *net.TCPAddr: %v", aListener.Addr())
	}
	bTCPAddr, ok := bListener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("bListener.Addr() is not a *net.TCPAddr: %v", aListener.Addr())
	}
	if aTCPAddr.Port == bTCPAddr.Port {
		t.Errorf("FreeTCPPortListener() got same port: %v, %v", aListener, bListener)
	}
	if aTCPAddr.Port == 0 || bTCPAddr.Port == 0 {
		t.Errorf("FreeTCPPortListener() got port 0")
	}
}
