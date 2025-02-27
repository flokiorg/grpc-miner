package sha256

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func sha256OT(block string) string {
	blockBytes, _ := hex.DecodeString(block)
	hashBytes := Sum256(blockBytes)
	return hex.EncodeToString(hashBytes[:])
}

func sha256DoublehashOT(block string) string {
	blockBytes, _ := hex.DecodeString(block)
	hashBytes := DoubleSum256(blockBytes)
	return hex.EncodeToString(hashBytes[:])
}

func TestSHA256OT(t *testing.T) {
	expected := "62f8b55856cdee6c262f6fe006a06475fbad12df3bbbc8a48f7066e39a7e9fb5"
	block := "010000004ddccd549d28f385ab457e98d1b11ce80bfea2c5ab93015ade4973e400000000bf4473e53794beae34e64fccc471dace6ae544180816f89591894e0f417a914cd74d6e49ffff001d323b3a7b"
	hash := sha256OT(block)

	if hash != expected {
		t.Fatalf("mismatch hash got=%s", hash)
	}

}

func TestSHA256DoubleHashOT(t *testing.T) {
	expected := "e770c2a77c47cfc24caf6edfeccbd4ef242269752e1da6b240a2c5b000000000"
	block := "010000004ddccd549d28f385ab457e98d1b11ce80bfea2c5ab93015ade4973e400000000bf4473e53794beae34e64fccc471dace6ae544180816f89591894e0f417a914cd74d6e49ffff001d323b3a7b"
	hash := sha256DoublehashOT(block)

	if hash != expected {
		t.Fatalf("mismatch hash got=%s", hash)
	}

}

func BenchmarkSHA256(b *testing.B) {

	block := "010000004ddccd549d28f385ab457e98d1b11ce80bfea2c5ab93015ade4973e400000000bf4473e53794beae34e64fccc471dace6ae544180816f89591894e0f417a914cd74d6e49ffff001d323b3a7b"
	blockBytes, _ := hex.DecodeString(block)

	for i := 0; i < b.N; i++ {
		DoubleSum256(blockBytes)
	}
}

func TestSHA256CT(t *testing.T) {

	block := "010000004ddccd549d28f385ab457e98d1b11ce80bfea2c5ab93015ade4973e400000000bf4473e53794beae34e64fccc471dace6ae544180816f89591894e0f417a914cd74d6e49ffff001d"
	blockBytes, _ := hex.DecodeString(block)

	nonces := []string{"323b3a7a", "323b3a7b", "323b3a7c", "323b3a7d", "323b3a7e", "323b3a7f"}
	expectedHashes := []string{}

	for _, nonce := range nonces {
		hash := sha256OT(fmt.Sprintf("%s%s", block, nonce))
		expectedHashes = append(expectedHashes, hash)
	}

	hash := New()
	hash.Write(blockBytes[:64])
	h, slen := hash.State()
	log.Debug().Msgf("len:%v h:%v", slen, h)

	for i := 0; i < len(nonces); i++ {
		nonceBytes, _ := hex.DecodeString(nonces[i])
		hash := New()
		hash.SetState(h, slen)

		suffixBytes := append(blockBytes[64:], nonceBytes...)
		log.Debug().Msgf("suffix: %v", hex.EncodeToString(suffixBytes))

		result := hash.Sum(suffixBytes)
		if h := hex.EncodeToString(result); h != expectedHashes[i] {
			t.Fatalf("mismatch hash got=%s", h)
		}
		t.Logf("nonce:%s hash:%s", nonces[i], expectedHashes[i])
	}

}
