// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *Server) csvFromFields(dst io.Writer, r *http.Request) error {
	writer := csv.NewWriter(dst)
	writer.UseCRLF = false
	writer.Comma = ';'
	values := r.MultipartForm.Value

	headers := make([]string, 0, len(values))
	for field := range values {
		headers = append(headers, field)
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	row := make([]string, len(headers))
	for i, field := range headers {
		row[i] = strings.Join(values[field], "\n")
	}
	if err := writer.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	writer.Flush()
	return writer.Error()
}
