package botsrv

import (
	"bytes"
	"text/template"
)

var tmplStudentCard = `Новая заявка от лицеиста!

{{if .Name}}Имя: {{.Name}} {{end}}

{{if .Class}}Класс: {{.Class}} {{end}}

{{if .Nickname}}Ник: @{{.Nickname}} {{end}}`

var tmplGraduateCard = `Новая заявка от выпускника!

{{if .Name}}Имя: {{.Name}} {{end}}

{{if .Year}}Год выпуска: {{.Year}} {{end}}

{{if .Class}}Класс: {{.Class}} {{end}}

{{if .CityInfo}}Города: {{.CityInfo}} {{end}}

{{if .UniversityInfo}}ВУЗы: {{.UniversityInfo}} {{end}}

{{if .WorkInfo}}Работа: {{.WorkInfo}} {{end}}

{{if .Nickname}}Ник: @{{.Nickname}} {{end}}`

func parseStudent(student StudentForm) (string, error) {
	tmpl, err := template.New("tmplStudentCard").Parse(tmplStudentCard)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, student)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func parseGraduate(grad GraduateForm) (string, error) {
	tmpl, err := template.New("tmplGraduateCard").Parse(tmplGraduateCard)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, grad)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}
