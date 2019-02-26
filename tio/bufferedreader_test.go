// Copyright 2015-2018 trivago N.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tio

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/trivago/tgo/ttesting"
)

type bufferedReaderTestData struct {
	expect ttesting.Expect
	tokens []string
	parsed int
}

func (br *bufferedReaderTestData) write(data []byte) {
	br.expect.Equal(br.tokens[br.parsed], string(data))
	br.parsed++
}

func TestBufferedReaderDelimiter(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := strings.Join(data.tokens, "\n")
	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, 0, 0, "\n")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.Equal(io.EOF, err)
	data.expect.Equal(2, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMultilineDelimiter(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{
			"1111-12-06 1\n",
			"2222-12-06T14:58:44.060 [qtp1860944798-11] a08b3652499144f4ac7bbf0bb12e012f ERROR portal2Service.App - \n",
			"java.sql.SQLTransientConnectionException: HikariPool-1 - Connection is not available, request timed out after 30013ms.\n",
			"  at com.zaxxer.hikari.pool.HikariPool.createTimeoutException(HikariPool.java:676)\n",
			"  at com.zaxxer.hikari.pool.HikariPool.getConnection(HikariPool.java:190)\n",
			"  ... 21 common frames omitted\n",
			"3333-12-06 3\n",
		},
		parsed: 0,
	}

	parseData := strings.Join(data.tokens, "")
	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, 32, 0, "^\\d{4}-\\d{2}-\\d{2}")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.Equal(io.EOF, err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLEText(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := ""
	for _, s := range data.tokens {
		parseData += fmt.Sprintf("%d:%s", len(s), s)
	}

	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE, 0, ":")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderFixed(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test2", "test3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLEFixed, 5, "")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLE8(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, byte(len(s)))
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE8, 0, "")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLE16(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, byte(len(s)), 0) // hacky but ok for little endian
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE16, 0, "")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLE32(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, byte(len(s)), 0, 0, 0) // hacky but ok for little endian
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE32, 0, "")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLE64(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, byte(len(s)), 0, 0, 0, 0, 0, 0, 0) // hacky but ok for little endian
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE64, 0, "")

	err := reader.ReadAll(parseReader, data.write)
	data.expect.NoError(err)
	data.expect.Equal(3, data.parsed)

	msg, _, err := reader.ReadOne(parseReader)
	data.expect.Equal(io.EOF, err)
	data.expect.Nil(msg)
}

func TestBufferedReaderMLE8EO(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	var parseData []byte
	for _, s := range data.tokens {
		parseData = append(parseData, 0, 0, byte(len(s)))
		parseData = append(parseData, s...)
	}

	parseReader := bytes.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE8|BufferedReaderFlagEverything, 2, "")

	offset := 0
	for _, s := range data.tokens {
		msg, _, err := reader.ReadOne(parseReader)
		data.expect.NoError(err)
		nextOffset := offset + 3 + len(s)
		data.expect.Equal(parseData[offset:nextOffset], msg)
		offset = nextOffset
	}
}

func TestBufferedReaderMLETextEO(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := ""
	for _, s := range data.tokens {
		parseData += fmt.Sprintf("  %d:%s", len(s), s)
	}

	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagMLE|BufferedReaderFlagEverything, 2, ":")

	for _, s := range data.tokens {
		msg, _, err := reader.ReadOne(parseReader)
		data.expect.NoError(err)
		data.expect.Equal(fmt.Sprintf("  %d:%s", len(s), s), string(msg))
	}
}

func TestBufferedReaderDelimiterE(t *testing.T) {
	data := bufferedReaderTestData{
		expect: ttesting.NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := strings.Join(data.tokens, "\n")
	parseData += "\n"
	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagEverything, 0, "\n")

	for _, s := range data.tokens {
		msg, _, err := reader.ReadOne(parseReader)
		data.expect.NoError(err)
		data.expect.Equal(fmt.Sprintf("%s\n", s), string(msg))
	}
}
