// Package tmplsrv defines a simple TemplateServer that treats a http.FileSystem
// passed to it as a repository of templates, and renders them using the data
// provided. The
//
// This is a simplistic implementation that very likely has a number of issues,
// considering separate file types are not even considered beyond trying to set
// a Content-Type header. E.g., escaping strategies in Javascript, CSS, and
// HTML files will differ.
package tmplsrv

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type (
	templateServer struct {
		fs       http.FileSystem
		staticFS http.FileSystem
		data     map[string]interface{}
		rendered map[string]rendered
	}

	rendered struct {
		tmplPath string
		content  io.Reader
		modtime  time.Time
	}
)

func (ts templateServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	upath = path.Clean(upath)
	log := logrus.WithField("file", upath)

	// Try static files first.
	//
	// NOTE Unable to serve index.html with this handler. There is
	// special behavior noted about index.html files being served,
	// so this needs to be looked into further.
	if f, err := ts.staticFS.Open(upath); err == nil {
		log.Debug("serving from static files")
		defer f.Close()
		io.Copy(rw, f)
		return
	}

	// Try pre-rendered templates that aren't outdated.
	if rendered, ok := ts.rendered[upath]; ok && rendered.IsCurrent(ts.fs) {
		log.Debug("serving from rendered template cache")
		setContentType(rw, upath)
		io.Copy(rw, rendered.content)
		return
	}

	// Render the template.
	rendered, err := ts.renderTemplate(upath)
	if os.IsNotExist(errors.Cause(err)) {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"template": upath,
			"err":      err,
		}).Info("failed parsing template")
		http.Error(rw, "Error generating page", http.StatusInternalServerError)
		return
	}

	logrus.Debug("rendered template")
	ts.rendered[upath] = rendered
	setContentType(rw, upath)
	io.Copy(rw, rendered.content)
}

func (ts templateServer) parseTemplate(tmplPath string) (*template.Template, error) {
	file, err := ts.fs.Open(tmplPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return template.New(tmplPath).Parse(string(contents))
}

func (ts templateServer) renderTemplate(tmplPath string) (rendered, error) {
	var buf bytes.Buffer
	tmpl, err := ts.parseTemplate(tmplPath)
	if err != nil {
		return rendered{}, err
	}
	if err := tmpl.Execute(&buf, ts.data); err != nil {
		return rendered{}, err
	}
	fileinfo, err := stat(ts.fs, tmplPath)
	if err != nil {
		return rendered{}, err
	}
	return rendered{
		tmplPath: fileinfo.Name(),
		content:  &buf,
		modtime:  fileinfo.ModTime(),
	}, nil
}

// IsCurrent reports whether a rendered template is current, with
// regard to the file in the filesystem. This allows the template
// content to change without having to restart the server.
func (r rendered) IsCurrent(fs http.FileSystem) bool {
	fileinfo, err := stat(fs, r.tmplPath)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": r.tmplPath,
			"err":      err,
		}).Errorf("cannot stat file %q", r.tmplPath)
		return false
	}
	return fileinfo.ModTime().Before(r.modtime)
}

func stat(fs http.FileSystem, filename string) (os.FileInfo, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

func setContentType(rw http.ResponseWriter, fileName string) {
	mimeType := mime.TypeByExtension(path.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	logrus.WithFields(logrus.Fields{
		"file_name": fileName,
		"mime_type": mimeType,
	}).Debug("determined content type")
	rw.Header().Set("Content-Type", mimeType)
}

func TemplateServer(staticFS, fs http.FileSystem, data map[string]interface{}) http.Handler {
	return templateServer{
		fs:       fs,
		staticFS: staticFS,
		data:     data,
		rendered: make(map[string]rendered),
	}
}
