package runner_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stretchr/testify/suite"
)

const gitLogs = "Cloning into 'sample-project'...\n" +
	"remote: Enumerating objects: 214, done.\n" +
	"remote: Counting objects:   0% (1/214)                    \r" +
	"remote: Counting objects:   1% (3/214)                    \r" +
	"remote: Counting objects:  99% (212/214)                  \r" +
	"remote: Counting objects: 100% (214/214), done.\n" +
	"remote: Compressing objects:   0% (1/135)                 \r" +
	"remote: Compressing objects:  97% (131/135)               \r" +
	"remote: Compressing objects: 100% (135/135), done.\n" +
	"Receiving objects:   0% (1/214)						   \r" +
	"Receiving objects:  50% (107/214)						   \r" +
	"Receiving objects:  91% (195/214), 556.00 KiB | 1.03 MiB/s						\r" +
	"remote: Total 214 (delta 110), reused 173 (delta 72), pack-reused 0 (from 0)\n" +
	"Receiving objects: 100% (214/214), 556.00 KiB | 1.03 MiB/s						\r" +
	"Receiving objects: 100% (214/214), 977.05 KiB | 1.34 MiB/s, done.\n" +
	"Resolving deltas:   0% (0/110)				\r" +
	"Resolving deltas: 100% (110/110)			\r" +
	"Resolving deltas: 100% (110/110), done."

const expectedGitLogs = "Cloning into 'sample-project'...\n" +
	"remote: Enumerating objects: 214, done.\n" +
	"remote: Counting objects: 100% (214/214), done.\n" +
	"remote: Compressing objects: 100% (135/135), done.\n" +
	"remote: Total 214 (delta 110), reused 173 (delta 72), pack-reused 0 (from 0)\n" +
	"Receiving objects: 100% (214/214), 977.05 KiB | 1.34 MiB/s, done.\n" +
	"Resolving deltas: 100% (110/110), done."

type CustomBufferSuite struct {
	suite.Suite
}

func (s *CustomBufferSuite) Test_Read_Write() {
	cb := runner.NewCustomBuffer()

	// Write hello world
	written, err := cb.Write([]byte("Hello World"))
	s.Nil(err)
	s.Equal(11, written)

	s.Equal("Hello World\n", cb.String())
}

func (s *CustomBufferSuite) Test_GitLogs() {
	cb := runner.NewCustomBuffer()

	// Write the whole lorem ipsum
	written, err := cb.Write([]byte(gitLogs))
	s.Nil(err)
	s.Equal(len(gitLogs), written)

	s.Equal(expectedGitLogs, cb.String()[:len(expectedGitLogs)])
}

func (s *CustomBufferSuite) Test_Write_CarriageReturn() {
	cb := runner.NewCustomBuffer()
	text := []byte("Let's see\nHello World\rHi World\rMy World")

	// Write hello world
	written, err := cb.Write(text)
	s.Nil(err)
	s.Equal(len(text), written)

	expected := "Let's see\nMy World"

	s.Equal(expected, cb.String()[:len(expected)])
}

func TestCustomBufferSuite(t *testing.T) {
	suite.Run(t, &CustomBufferSuite{})
}
