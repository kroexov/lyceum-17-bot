package botsrv

import (
	"bytes"
	"strings"
	"text/template"
	"unicode"
)

var tmplStudentCard = `Новая заявка от лицеиста!

{{if .Name}}Имя: {{.Name}} {{end}}

{{if .Class}}Класс: {{.Class}} {{end}}

{{if .Nickname}}Ник: @{{.Nickname}} {{end}}`

var tmplGraduateCard = `Новая заявка от выпускника!

{{if .Name}}Имя: {{.Name}} {{end}}

{{if .Year}}Выпуск {{.Year}}, {{if .Class}}{{.Class}} класс{{end}}{{end}}

{{if .CityInfo}}Города: {{.CityInfo}} {{end}}

{{if .UniversityInfo}}ВУЗы: {{.UniversityInfo}} {{end}}

{{if .WorkInfo}}Работа: {{.WorkInfo}} {{end}}

{{if .ExtraInfo}}Дополнительно о себе: {{.ExtraInfo}} {{end}}

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

	res := result.String()

	var hashtags []string
	for _, s := range strings.Split(grad.CityInfo, ",") {
		s = KeepAllowedChars(strings.TrimSpace(strings.ToLower(s)))
		hashtags = append(hashtags, "#"+s)
	}
	for _, s := range strings.Split(grad.UniversityInfo, ",") {
		s = KeepAllowedChars(strings.TrimSpace(strings.ToLower(s)))
		hashtags = append(hashtags, "#"+s)
	}

	res = res + "\n" + strings.Join(hashtags, " ")

	return res, nil
}

func KeepAllowedChars(input string) string {
	var result []rune
	for _, char := range input {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' {
			result = append(result, char)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
