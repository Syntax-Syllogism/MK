package board

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"
)

type Task struct {
	File      string   `json:"file"`
	Status    string   `yaml:"status" json:"status"`
	Project   string   `yaml:"project" json:"project"`
	Epic      string   `yaml:"epic" json:"epic"`
	Order     *float64 `yaml:"order,omitempty" json:"order,omitempty"`
	Tags      []string `yaml:"tags" json:"tags"`
	DependsOn []string `yaml:"depends_on" json:"dependsOn,omitempty"`
	Title     string   `json:"title"`
	Heading   string   `yaml:"-" json:"heading,omitempty"`
	Content   string   `json:"content"`
	// ParseError is set only when the file's frontmatter failed to parse; it
	// is never read from or written to YAML. Content still holds the file's
	// real body in this case, so opening the card and saving it back (e.g.
	// via the tag editor) round-trips safely instead of clobbering the body
	// with an error message.
	ParseError string `json:"parseError,omitempty"`
}

const (
	StatusConflict = "CONFLICT"
	StatusOther    = "OTHER"
	NoEpic         = "NO EPIC"
)

func ParseTask(path string) (Task, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return Task{}, err
	}

	task := Task{
		File:  filepath.Base(path),
		Title: strings.TrimSuffix(filepath.Base(path), ".md"),
	}

	// Check for Git conflict markers
	if strings.Contains(string(content), "<<<<<<<") {
		task.Status = StatusConflict
		task.Content = string(content)
		return task, nil
	}

	// Basic frontmatter parsing. Require the file to actually start with the
	// delimiter — SplitN alone would treat any two "---" lines anywhere in the
	// body as a frontmatter block (F25), misparsing plain notes that use "---"
	// as a thematic break.
	parts := strings.SplitN(string(content), "---", 3)
	if strings.HasPrefix(string(content), "---") && len(parts) >= 3 {
		var fm Task
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil ||
			(fm.Order != nil && (math.IsNaN(*fm.Order) || math.IsInf(*fm.Order, 0))) {
			// Malformed frontmatter (e.g. `order: soon`, or a scalar where a
			// list is expected) must not make the task disappear. Leave
			// task.Status empty so it lands in OTHER, and surface the error
			// via ParseError (not Epic — Epic feeds the filter bar's chip
			// list, and a real-looking "PARSE ERROR" epic would pollute it)
			// rather than Content — Content must stay the file's real body,
			// or opening the card and saving it back (e.g. via the tag
			// editor, which doesn't touch the textarea) would silently
			// overwrite that body with the error message.
			if err != nil {
				task.ParseError = err.Error()
			} else {
				task.ParseError = "order must be a finite number"
			}
			task.Content = strings.TrimSpace(parts[2])
			task.Heading = firstHeading(task.Content)
			return task, nil
		}
		task.Status = fm.Status
		task.Project = fm.Project
		task.Epic = fm.Epic
		task.Order = fm.Order
		task.Tags = fm.Tags
		task.DependsOn = fm.DependsOn
		task.Content = strings.TrimSpace(parts[2])
	} else {
		task.Content = strings.TrimSpace(string(content))
	}

	task.Heading = firstHeading(task.Content)
	return task, nil
}

func firstHeading(body string) string {
	var fence byte
	fenceLength := 0
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			marker := trimmed[0]
			length := len(trimmed) - len(strings.TrimLeft(trimmed, string(marker)))
			if length >= 3 {
				if fence == 0 {
					fence, fenceLength = marker, length
				} else if marker == fence && length >= fenceLength && strings.TrimSpace(trimmed[length:]) == "" {
					fence, fenceLength = 0, 0
				}
				continue
			}
		}
		if fence != 0 || !strings.HasPrefix(line, "# ") {
			continue
		}
		heading := strings.TrimSpace(line[2:])
		withoutHashes := strings.TrimRight(heading, "#")
		if withoutHashes != heading && (withoutHashes == "" || strings.HasSuffix(withoutHashes, " ") || strings.HasSuffix(withoutHashes, "\t")) {
			heading = strings.TrimSpace(withoutHashes)
		}
		if heading != "" {
			return heading
		}
	}
	return ""
}

// FrontmatterKeys returns raw top-level YAML nodes, distinguishing missing keys
// from explicit empty values. Its delimiter rules match ParseTask.
func FrontmatterKeys(path string) (map[string]yaml.Node, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := map[string]yaml.Node{}
	parts := strings.SplitN(string(data), "---", 3)
	if !strings.HasPrefix(string(data), "---") || len(parts) < 3 {
		return result, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(parts[1]), &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return result, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, ErrInvalidFrontmatter
	}
	for i := 0; i < len(root.Content); i += 2 {
		result[root.Content[i].Value] = *root.Content[i+1]
	}
	return result, nil
}
