# RWTH.video scraper

Proof of Concept scraper for recorded lectures from RWTH Aachen University.

Upon running scrapes all courses and each episode into a single [courses.json](courses.json)

## Downloading

See [jq](https://jqlang.github.io/jq/) and [curl](https://curl.se/)

# Dataformat

```golang
type Course struct {
	Id       string
	Author   Author
	Title    string
	Episodes map[string]Episode
	Feed     string
	Updated  string
}

type Author struct {
	Name  string
	Email string
}

type Episode struct {
	Id       string
	Title    string
	Updated  string
	Summary  string
	Files    []string
	Chapters string
}
```
