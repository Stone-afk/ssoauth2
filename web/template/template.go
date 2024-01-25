package template

import (
	"bytes"
	"context"
	"html/template"
	"io/fs"
)

type TemplateEngine interface {
	// Render 渲染页面
	// data 是渲染页面所需要的数据
	ExcuteTpl() ([]byte, error)
	Render(ctx context.Context, tplName string, data any) ([]byte, error)
	LoadFromGlob(pattern string) error
	LoadFromFiles(filenames ...string) error
	LoadFromFS(fs fs.FS, filenames ...string) error
}

type GoTemplateEngine struct {
	T *template.Template
	// 也可以考虑设计为 map[string]*template.Template
	// 但是其实没太大必要，因为 template.Template 本身就提供了按名索引的功能
}

func (t *GoTemplateEngine) ExcuteTpl() ([]byte, error) {
	var err error
	buffer := &bytes.Buffer{}
	err = t.T.Execute(buffer, nil)
	return buffer.Bytes(), err
}

func (t *GoTemplateEngine) Render(ctx context.Context, tplName string, data any) ([]byte, error) {
	res := &bytes.Buffer{}
	err := t.T.ExecuteTemplate(res, tplName, data)
	return res.Bytes(), err
}

//  以下是管理模板的方法, Web 框架根本不在意你从哪里把 模板搞到，
// 它只关心 Render 方法你要实现，所以说 管理模板的方法并不算是 TemplateEngine 接口的一 部分

func (t *GoTemplateEngine) LoadFromGlob(pattern string) error {
	var err error
	t.T, err = template.ParseGlob(pattern)
	return err
}

func (t *GoTemplateEngine) LoadFromFiles(filenames ...string) error {
	var err error
	t.T, err = template.ParseFiles(filenames...)
	return err
}

func (t *GoTemplateEngine) LoadFromFS(fs fs.FS, filenames ...string) error {
	var err error
	t.T, err = template.ParseFS(fs, filenames...)
	return err
}
