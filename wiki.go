package main

import (
  "regexp"
  "os"
  "html/template"
  "io/ioutil"
  "net/http"
  "github.com/russross/blackfriday"
)

//
// Page
//
type Page struct {
  Title string
  Body []byte
}

func (p *Page) save() error {
  filename := "data/" + p.Title + ".txt"
  return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
  filename := "data/" + title + ".txt"
  body, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  return &Page{Title: title, Body: body}, nil
}

//
// Template
//
func markdown(input []byte) []byte {
  htmlFlags := 0
  htmlFlags |= blackfriday.HTML_SKIP_HTML
  render := blackfriday.HtmlRenderer(htmlFlags, "", "")

  extensions := 0
  extensions |= blackfriday.EXTENSION_FENCED_CODE
  extensions |= blackfriday.EXTENSION_AUTOLINK

  return blackfriday.Markdown(input, render, extensions)
}

func unsafe(str string) template.HTML {
  return template.HTML(str)
}

type TemplateMap map[string]*template.Template

func prepareTemplates(filenames ...string) TemplateMap {
  funcMap := template.FuncMap {
    "markdown": markdown,
    "unsafe": unsafe,
  }
  tmpls := make(TemplateMap)
  for _, filename := range filenames {
    files := []string{"views/" + filename, "views/layout.html"}
    tmpls[filename] = template.Must(template.New("").Funcs(funcMap).ParseFiles(files...))
  }
  return tmpls
}

var templates = prepareTemplates("edit.html", "view.html")

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
  err := templates[tmpl + ".html"].ExecuteTemplate(w, "layout", p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
}

//
// Handlers
//
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
    fn(w, r, m[2])
  }
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    http.Redirect(w, r, "/edit/" + title, http.StatusFound)
    return
  }
  renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }
  renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
  body := r.FormValue("body")
  p := &Page{Title: title, Body: []byte(body)}
  err := p.save()
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path == "/" {
    http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
  } else {
    http.NotFound(w, r)
  }
}

func main() {
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.HandleFunc("/", rootHandler)

  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  http.ListenAndServe(":" + port, nil)
}

