package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File represents a single editable config file. Unlike Config (which merges
// several files for reading), File operates on exactly one file and can write
// changes back to disk, preserving existing key order and comments.
type File struct {
	path string
	doc  *yaml.Node
}

// newEmptyDoc returns a document node wrapping an empty mapping.
func newEmptyDoc() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map"},
		},
	}
}

// LoadFile reads a single config file. A missing file yields an empty File that
// can still be modified and saved.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{path: path, doc: newEmptyDoc()}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return &File{path: path, doc: newEmptyDoc()}, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parsing %s: expected a mapping of account names", path)
	}
	return &File{path: path, doc: &doc}, nil
}

// Path returns the file's path.
func (f *File) Path() string { return f.path }

// root returns the top-level mapping node.
func (f *File) root() *yaml.Node {
	return f.doc.Content[0]
}

// Names returns the account names in file order.
func (f *File) Names() []string {
	r := f.root()
	names := make([]string, 0, len(r.Content)/2)
	for i := 0; i+1 < len(r.Content); i += 2 {
		names = append(names, r.Content[i].Value)
	}
	return names
}

// Accounts returns all accounts in file order.
func (f *File) Accounts() ([]Account, error) {
	r := f.root()
	accts := make([]Account, 0, len(r.Content)/2)
	for i := 0; i+1 < len(r.Content); i += 2 {
		var a Account
		if err := r.Content[i+1].Decode(&a); err != nil {
			return nil, fmt.Errorf("decoding account %q: %w", r.Content[i].Value, err)
		}
		a.Name = r.Content[i].Value
		accts = append(accts, a)
	}
	return accts, nil
}

// Has reports whether an account with the given name exists.
func (f *File) Has(name string) bool {
	r := f.root()
	for i := 0; i+1 < len(r.Content); i += 2 {
		if r.Content[i].Value == name {
			return true
		}
	}
	return false
}

// Set adds or replaces an account. The account's Name is used as the key.
func (f *File) Set(a Account) error {
	valNode := &yaml.Node{}
	if err := valNode.Encode(a); err != nil {
		return fmt.Errorf("encoding account %q: %w", a.Name, err)
	}

	r := f.root()
	for i := 0; i+1 < len(r.Content); i += 2 {
		if r.Content[i].Value == a.Name {
			r.Content[i+1] = valNode
			return nil
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: a.Name}
	r.Content = append(r.Content, keyNode, valNode)
	return nil
}

// Remove deletes an account by name, reporting whether it existed.
func (f *File) Remove(name string) bool {
	r := f.root()
	for i := 0; i+1 < len(r.Content); i += 2 {
		if r.Content[i].Value == name {
			r.Content = append(r.Content[:i], r.Content[i+2:]...)
			return true
		}
	}
	return false
}

// Save writes the file back to disk, creating parent directories as needed and
// using owner-only permissions since the file holds plaintext passwords.
func (f *File) Save() error {
	if dir := filepath.Dir(f.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	data, err := yaml.Marshal(f.doc)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(f.path, data, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", f.path, err)
	}
	return nil
}
