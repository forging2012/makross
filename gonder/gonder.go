package gonder

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/insionng/makross"
)

type (
	Renderer struct {
		Option
		templates map[string]*template.Template
		lock      sync.RWMutex
	}

	Option struct {
		// Directory to load templates. Default is "templates"
		Directory string
		// Reload to reload templates everytime.
		Reload bool
		// Filter to do Filter for templates
		Filter bool
		// DelimLeft "{{"
		DelimLeft string
		// DelimRight "}}"
		DelimRight string
	}
)

func perparOption(options []Option) Option {
	var opt Option
	if len(options) > 0 {
		opt = options[0]
	}
	if len(opt.Directory) == 0 {
		opt.Directory = "template"
	}
	if len(opt.DelimLeft) == 0 {
		opt.DelimLeft = "{{"
	}
	if len(opt.DelimRight) == 0 {
		opt.DelimRight = "}}"
	}
	return opt
}

func Renderor(opt ...Option) *Renderer {
	o := perparOption(opt)
	r := &Renderer{
		Option:    o,
		templates: make(map[string]*template.Template),
	}
	return r
}

func (r *Renderer) buildTemplatesCache(name string) (t *template.Template, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	t, err = template.ParseFiles(filepath.Join(r.Directory, name))
	if err != nil {
		return
	}
	r.templates[name] = t
	return
}

func (r *Renderer) getTemplate(name string) (t *template.Template, err error) {
	name = name + ".html"
	if r.Reload {
		return template.ParseFiles(filepath.Join(r.Directory, name))
	}
	r.lock.RLock()
	var okay bool
	if t, okay = r.templates[name]; !okay {
		r.lock.RUnlock()
		t, err = r.buildTemplatesCache(name)
	} else {
		r.lock.RUnlock()
	}
	return
}

// Render 渲染
func (r *Renderer) Render(w io.Writer, name string, c *makross.Context) (err error) {
	template, err := r.getTemplate(name)
	if err != nil {
		return err
	}
	template.Delims(r.DelimLeft, r.DelimRight)

	var buffer bytes.Buffer
	err = template.Execute(&buffer, c.GetStore())
	if err != nil {
		return err
	}

	if b := buffer.Bytes(); r.Filter {
		_, err = fmt.Fprintf(w, "%s", c.DoFilterHook(fmt.Sprintf("%s_template", name), func() []byte {
			return b
		}))
	} else {
		_, err = fmt.Fprintf(w, "%s", b)
	}
	return err

}
