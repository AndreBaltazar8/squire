package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/AndreBaltazar8/squire/internal/config"
	"github.com/AndreBaltazar8/squire/internal/project"
)

type componentPickerItem struct {
	ID          string
	Description string
}

func interactiveComponents(cmd *cobra.Command, cwd string, components []config.Component, selected []string) ([]string, error) {
	if len(components) == 0 {
		return selected, nil
	}
	preselected := selected
	if preselected == nil {
		preselected = project.DetectedComponentIDs(cwd, components)
	}
	return runComponentPicker(cmd.InOrStdin(), cmd.ErrOrStderr(), components, preselected)
}

func runComponentPicker(input io.Reader, output io.Writer, components []config.Component, preselected []string) ([]string, error) {
	inFile, ok := input.(*os.File)
	if !ok {
		inFile = os.Stdin
	}
	fd := int(inFile.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("--interactive requires terminal stdin")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)
	defer fmt.Fprint(output, "\x1b[?25h\x1b[2J\x1b[H")

	selected := map[string]bool{}
	for _, id := range preselected {
		id = strings.TrimSpace(id)
		if id != "" {
			selected[id] = true
		}
	}

	query := ""
	cursor := 0
	reader := bufio.NewReader(inFile)
	fmt.Fprint(output, "\x1b[?25l")

	for {
		filtered := filteredComponents(components, query)
		if cursor >= len(filtered) {
			cursor = len(filtered) - 1
		}
		if cursor < 0 {
			cursor = 0
		}

		drawComponentPicker(output, fd, filtered, selected, query, cursor)

		key, err := readPickerKey(reader, fd)
		if err != nil {
			return nil, err
		}

		switch {
		case key.cancel:
			return nil, errors.New("interactive selection canceled")
		case key.submit:
			return selectedComponentResult(components, selected), nil
		case key.up:
			if cursor > 0 {
				cursor--
			}
		case key.down:
			if cursor < len(filtered)-1 {
				cursor++
			}
		case key.toggle:
			if len(filtered) > 0 {
				id := filtered[cursor].ID
				selected[id] = !selected[id]
			}
		case key.backspace:
			if query != "" {
				query = query[:len(query)-1]
				cursor = 0
			}
		case key.char != 0:
			query += string(key.char)
			cursor = 0
		}
	}
}

type pickerKey struct {
	char      byte
	up        bool
	down      bool
	toggle    bool
	submit    bool
	backspace bool
	cancel    bool
}

func readPickerKey(reader *bufio.Reader, fd int) (pickerKey, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return pickerKey{}, err
	}

	switch b {
	case 3:
		return pickerKey{cancel: true}, nil
	case 13, 10:
		return pickerKey{submit: true}, nil
	case 8, 127:
		return pickerKey{backspace: true}, nil
	case ' ':
		return pickerKey{toggle: true}, nil
	case 27:
		if !inputReady(reader, fd, 40*time.Millisecond) {
			return pickerKey{cancel: true}, nil
		}
		next, err := reader.ReadByte()
		if err != nil {
			return pickerKey{}, err
		}
		if !inputReady(reader, fd, 40*time.Millisecond) {
			return pickerKey{}, nil
		}
		final, err := reader.ReadByte()
		if err != nil {
			return pickerKey{}, err
		}
		if next == '[' {
			switch final {
			case 'A':
				return pickerKey{up: true}, nil
			case 'B':
				return pickerKey{down: true}, nil
			}
		}
		return pickerKey{}, nil
	}

	if b >= 33 && b <= 126 {
		return pickerKey{char: b}, nil
	}
	return pickerKey{}, nil
}

func inputReady(reader *bufio.Reader, fd int, timeout time.Duration) bool {
	if reader.Buffered() > 0 {
		return true
	}
	pollFDs := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(pollFDs, int(timeout/time.Millisecond))
	return err == nil && n > 0 && pollFDs[0].Revents&unix.POLLIN != 0
}

func drawComponentPicker(output io.Writer, fd int, items []componentPickerItem, selected map[string]bool, query string, cursor int) {
	width, height, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		width = 100
	}
	if err != nil || height <= 0 {
		height = 24
	}
	maxRows := height - 7
	if maxRows < 5 {
		maxRows = 5
	}

	start := 0
	if cursor >= maxRows {
		start = cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	fmt.Fprint(output, "\x1b[2J\x1b[H")
	line(output, "Select Squire components")
	line(output, "Type to search. Space toggles. Enter generates. Esc cancels.")
	line(output, "Search: "+query)
	line(output, "")

	if len(items) == 0 {
		line(output, "No matching components.")
		return
	}

	for i := start; i < end; i++ {
		item := items[i]
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		box := "[ ]"
		if selected[item.ID] {
			box = "[x]"
		}
		text := prefix + box + " " + item.ID
		if item.Description != "" {
			text += " - " + strings.TrimSuffix(item.Description, ".")
		}
		line(output, truncate(text, width))
	}
	line(output, "")
	line(output, fmt.Sprintf("%d selected", countSelected(selected)))
}

func line(output io.Writer, text string) {
	fmt.Fprint(output, text+"\r\n")
}

func filteredComponents(components []config.Component, query string) []componentPickerItem {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []componentPickerItem
	for _, component := range components {
		id := strings.TrimSpace(component.ID)
		if id == "" {
			continue
		}
		description := strings.TrimSpace(component.Description)
		haystack := strings.ToLower(id + " " + description)
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		out = append(out, componentPickerItem{ID: id, Description: description})
	}
	return out
}

func selectedComponentResult(components []config.Component, selected map[string]bool) []string {
	var ids []string
	for _, component := range components {
		if selected[component.ID] {
			ids = append(ids, component.ID)
		}
	}
	return ids
}

func countSelected(selected map[string]bool) int {
	count := 0
	for _, value := range selected {
		if value {
			count++
		}
	}
	return count
}

func truncate(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
