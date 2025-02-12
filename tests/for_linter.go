package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

// необработанная ошибка (проверит `errcheck`)
func badErrorHandling() {
	file, _ := os.Open("nonexistent.txt") // Ошибка игнорируется
	fmt.Println(file)                     // Используем переменную, но файл не открыт
}

func insecureFileAccess() {
	// чтение файла в переменную (G304)
	content, _ := ioutil.ReadFile("/etc/shadow") // Небезопасный путь!
	fmt.Println(string(content))
}

// стиль кода (проверит `revive`)
func badStyle() {
	x := 10 // Неправильные пробелы вокруг (goland правит, можно отрубить)
	fmt.Println(x)
}

func main() {
	badErrorHandling()
	insecureFileAccess()
	badStyle()
}
