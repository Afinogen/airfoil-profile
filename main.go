package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/yofu/dxf"
	"github.com/yofu/dxf/color"
	"github.com/yofu/dxf/entity"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

func preRunCheck() {
	_, err := os.Stat("./data")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Print("Программа запускается в папке где ранее не запускалась. Создать папку data для хранения данных? [y/n]: ")
			var answer string
			fmt.Scanf("%s\n", &answer)
			if answer != "y" {
				fmt.Println("Для работы программы необходимо создать папку data. Переместите программу в директорию, где она может это сделать")
				os.Exit(0)
			}
			os.MkdirAll("./data/airfoil", 0755)
			os.MkdirAll("./data/output", 0755)
		}
	}
}

func getUrlContent(url string) (string, error) {

	resp, err := http.Get("http://airfoiltools.com" + url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = fmt.Errorf("ошибка в ответе сервера: %d %s", resp.StatusCode, resp.Status)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

type ProfileInfo struct {
	Title      string
	Link       string
	Content    string
	IsLocal    bool
	ChordWidth int32
	Thickness  int32
	MaxX       int32
	MaxY       int32
}

type Coordinate struct {
	X float32
	Y float32
}

func getLocalProfiles() (result []*ProfileInfo) {
	files, err := ioutil.ReadDir("./data/airfoil")
	if err != nil {
		log.Fatal(err)
	}

	if len(files) > 15 {
		//TODO Сделать поиск
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := path.Ext(file.Name())
			if ext == ".dat" {
				item := ProfileInfo{
					Title:   strings.ReplaceAll(file.Name(), ext, ""),
					Link:    "./data/airfoil/" + file.Name(),
					IsLocal: true,
				}
				result = append(result, &item)
			}
		}
	}
	return
}

func getExternalProfiles() (results []*ProfileInfo) {
	var profileName string
	fmt.Println("Поиск профилей идет по базе airfoiltools.com")
	for {
		fmt.Print("Введите название профиля на латинице: ")
		fmt.Scanf("%s\n", &profileName)

		if len(profileName) > 0 {
			break
		}
	}
	fmt.Println("Загрузка данных")

	results = searchOnSite(profileName)
	return
}

func selectProfile(results []*ProfileInfo, isLocal bool) int {
	if len(results) == 0 {
		fmt.Println("По запросу ничего не найдено")
		return -1
	}
	if !isLocal {
		fmt.Println("Результаты поиска:")
	}
	for i, result := range results {
		fmt.Println("[", i, "] ", formatProfileName(result.Title))
	}

	if isLocal {
		fmt.Println("Введите -1 если необходим поиск по базе airfoiltools")
	}
	fmt.Print("Введите номер профиля: ")
	var number int
	fmt.Scanf("%d\n", &number)
	if len(results) < number {
		fmt.Println("Неверный номер")
		return -1
	}

	return number
}

func formatProfileName(name string) string {
	name = strings.ReplaceAll(name, ".dat", "")
	name = strings.ReplaceAll(name, "AIRFOIL", "")
	name = strings.Split(name, ")")[1]
	name = strings.Trim(name, " ")

	return name
}

func searchOnSite(profileName string) (result []*ProfileInfo) {
	page := 0
	for {
		url := "/search/index?m[textSearch]=" + profileName + "&m[page]=" + strconv.Itoa(page)
		data, err := getUrlContent(url)

		if err != nil {
			log.Fatal(err)
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
		if err != nil {
			log.Fatal(err)
		}
		countResults := 0

		doc.Find("table.afSearchResult tbody tr").Each(func(i int, s *goquery.Selection) {
			title := s.Find("td.cell12 h3").Text()
			if len(title) > 0 {
				var link string
				s.Find("td.cell3 a").Each(func(i int, s *goquery.Selection) {
					if s.Text() == "Lednicer format dat file" {
						link, _ = s.Attr("href")
					}
				})
				item := ProfileInfo{Title: title, Link: link}
				result = append(result, &item)
				countResults += 1
			}
		})
		if countResults == 0 || page > 9 {
			break
		}
		page += 1
	}

	return result
}

func saveToFile(path string, content string) {
	f, err := os.Create(path)

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(content)

	if err2 != nil {
		log.Fatal(err2)
	}
}

func stringToFloat32(str string) (result float32) {
	value, err := strconv.ParseFloat(str, 32)
	if err != nil {
		log.Fatal(err.Error())
	}
	result = float32(value)
	return
}

func stringToInt(str string) int32 {
	value, err := strconv.Atoi(str)
	if err != nil {
		log.Fatal(err.Error())
	}
	return int32(value)

}

func getCoordinatesFromFile(profile *ProfileInfo) (coordinates []*Coordinate) {
	profileFile, err := ioutil.ReadFile("./data/airfoil/" + profile.Title + ".dat")

	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(profileFile), "\n")
	restrictions := strings.Split(lines[1], ".")
	profile.MaxX = stringToInt(strings.Trim(restrictions[0], " \t"))
	profile.MaxY = stringToInt(strings.Trim(restrictions[1], " \t"))

	for i := 3; i < len(lines); i++ {
		line := strings.Trim(lines[i], " \n\r")
		if line != "" {
			line = strings.ReplaceAll(line, "  ", " ")
			items := strings.Split(line, " ")

			c := Coordinate{
				X: stringToFloat32(items[0]),
				Y: stringToFloat32(items[1]),
			}
			coordinates = append(coordinates, &c)
		}
	}

	return
}

func prepareProfile(profile *ProfileInfo) {
	fmt.Printf("Выбран профиль: %s\n", formatProfileName(profile.Title))
	if !profile.IsLocal {
		profileConfig, err := getUrlContent(profile.Link)
		if err != nil {
			log.Fatal(err)
		}
		profile.Content = profileConfig
		saveToFile("./data/airfoil/"+profile.Title+".dat", profile.Content)
		fmt.Println("Профиль сохранен локально")
	}

	fmt.Println("Создание своего профиля, для выхода из программы нажмите Ctrl+C")
	for {
		fmt.Print("Введите ширну хорды (крыла) в мм: ")
		fmt.Scanf("%d\n", &profile.ChordWidth)
		fmt.Print("Введите высоту (толщину) профиля в мм (0 или пусто если оставить оригинал): ")
		fmt.Scanf("%d\n", &profile.Thickness)
		coordinates := getCoordinatesFromFile(profile)

		for _, coordinate := range coordinates {
			coordinate.X *= float32(profile.ChordWidth)
			coordinate.Y *= float32(profile.ChordWidth)
		}

		if profile.Thickness > 0 {
			maxThickness := float32(0)

			for _, coordinate := range coordinates {
				if coordinate.Y > maxThickness {
					maxThickness = coordinate.Y
				}
			}
			percent := float32(profile.Thickness) * 100 / maxThickness
			for _, coordinate := range coordinates {
				coordinate.Y *= percent / 100
			}
		}

		saveCoordinatesToCsv(coordinates, profile)
		saveCoordinatesToDxf(coordinates, profile)
	}
}

func saveCoordinatesToCsv(coordinates []*Coordinate, profile *ProfileInfo) {
	content := ""

	for i := profile.MaxX - 1; i >= 0; i-- {
		content += fmt.Sprintf("%.6f,%.6f\n", coordinates[i].X, coordinates[i].Y)
	}
	for i := profile.MaxX + 1; i <= profile.MaxX+profile.MaxY-1; i++ {
		content += fmt.Sprintf("%.6f,%.6f\n", coordinates[i].X, coordinates[i].Y)
	}

	fileName := fmt.Sprintf("./data/output/%s_%d_%d.csv", formatProfileName(profile.Title), profile.ChordWidth, profile.Thickness)
	saveToFile(fileName, content)
	fmt.Println("CSV сохранен в " + fileName)
}

func saveCoordinatesToDxf(coordinates []*Coordinate, profile *ProfileInfo) {
	d := dxf.NewDrawing()
	d.Header().LtScale = 100.0
	d.AddLayer("Profile", color.Grey192, dxf.DefaultLineType, true)

	p := entity.NewPolyline()
	p.SetLayer(d.CurrentLayer)

	for i := profile.MaxX - 1; i >= 0; i-- {
		p.AddVertex(float64(coordinates[i].X), float64(coordinates[i].Y), 0)
	}
	for i := profile.MaxX + 1; i <= profile.MaxX+profile.MaxY-1; i++ {
		p.AddVertex(float64(coordinates[i].X), float64(coordinates[i].Y), 0)
	}
	p.Close()
	d.AddEntity(p)

	fileName := fmt.Sprintf("./data/output/%s_%d_%d.dxf", formatProfileName(profile.Title), profile.ChordWidth, profile.Thickness)
	d.SaveAs(fileName)
	fmt.Println("DXF сохранен в " + fileName)
}

func main() {
	fmt.Println("Рассчет профиля крыла")
	preRunCheck()

	isLocal := true
	results := getLocalProfiles()
	if len(results) == 0 {
		results = getExternalProfiles()
		isLocal = false
	} else {
		fmt.Println("Ранее сохраненные профили:")
	}

	number := selectProfile(results, isLocal)
	if number == -1 && isLocal {
		isLocal = false
		results = getExternalProfiles()
		number = selectProfile(results, isLocal)
	}
	if number == -1 && !isLocal {
		return
	}

	prepareProfile(results[number])
}
