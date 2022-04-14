package main

import "fmt"
import "os"
import "log"
import "strings"
import "strconv"
import "github.com/eiannone/keyboard"
import "reflect"
import "bufio"
import "net/http"
import "io"
import "encoding/json"
import "crypto/md5"
import "encoding/base64"
import "gopkg.in/yaml.v2"

var cursor []int = []int{0, 0}
var args []string = os.Args[1:]
var userhome string = makeHome()

var col_reset = "\033[0m"
// --
var col_red = "\033[91m"
var col_yellow = "\033[93m"
var col_green = "\033[92m"
var col_cyan = "\033[96m"
var col_black = "\033[30m"
// --
var colb_red = "\033[41m"
var colb_cyan = "\033[46m"
var colb_black = "\033[40m"
var colb_yellow = "\033[103m"
var colb_green = "\033[42m"

var redraw = 0
var win = 0
var finish = 0
var config = loadConfig(userhome + "/.config/nonovi/conf.yaml")

type Config struct {
	NnvsDirectory string `yaml:"nnvsdir"`
	Remote string `yaml:"remote"`
}

func loadConfig(fp string) (*Config) {
	config := &Config{}
	_, err := os.Stat(fp)
	if os.IsNotExist(err) == true {
		config.NnvsDirectory = "/.config/nonovi/nnvs/"
		config.Remote = ""
		return config
	}
    f, err := os.ReadFile(fp)
    if err != nil { log.Fatal(err) }
    err = yaml.Unmarshal([]byte(f), &config)
    if err != nil { log.Fatal(err) }
    return config
}

func makeHome() string {
	userhome, err := os.UserHomeDir()
	if err != nil { log.Fatal(err) }
	err = os.MkdirAll(userhome + "/.config/nonovi", os.ModePerm)
	if err != nil { log.Fatal(err) }
	return userhome
}

func makeNnvsDirectory() {
	err := os.MkdirAll(userhome + config.NnvsDirectory, os.ModePerm)
	if err != nil { log.Fatal(err) }
}

func findIndex(s []string, p string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == p {
			return i
		}
	}
	return -1
}

func readDirectory(root string) ([]string, error) {
    var files []string
    f, err := os.Open(root)
    if err != nil { return files, err }
    fileInfo, err := f.Readdir(-1)
    f.Close()
    if err != nil { return files, err }

    for _, file := range fileInfo {
        files = append(files, file.Name())
    }
    return files, nil
}

type NonoviBoard struct {
    board [][]int //board is a slice of slices, first index is the y and the second x for board coords; left top corner is 0,0
    hints [][][]int //hints is a slice of slices of slices, it contains 2 slices - 1 for each axis - which contains slices for each row / column that contains hints for it
    lhr int //longest hint row
    lhc int //longest hint column
    mhr int //most hints row
    mhc int //most hints column
}

type nnvsResponse struct {
	Nnvs [][]string `json:"nnvs"`
}

type nnvResponse struct {
	Nnv []string `json:"nnv"`
}

type Nnv struct {
	Name string
	Md5 string
	Board string
}

func (b *NonoviBoard) loadBoard(fp string) {
	// reading input file
    content, err := os.ReadFile(fp)
    if err != nil { return }
    
	// reading and creating board attr
    con_spl := strings.Split(string(content), "\n")
    fb_ind := findIndex(con_spl, "b-")
    lb_ind := findIndex(con_spl, "eb-")
    con_board := con_spl[fb_ind + 1:lb_ind]
    b.board = make([][]int, len(con_board))
    for y:= 0; y < len(con_board); y++ {
        b.board[y] = make([]int, len(con_board[0]))
        for x:= 0; x < len(con_board[0]); x++ {
        	if string(con_board[y][x]) == "x" {
            	b.board[y][x] = 1
            } else {
            	b.board[y][x] = 0
            }
        }
    } 
    
	// reading and creating hints attr
    fh_index := findIndex(con_spl, "h-")
    lh_index := findIndex(con_spl, "eh-")
    con_hints := con_spl[fh_index + 1:lh_index]
    b.hints = make([][][]int, len(con_hints))
    for i := 0; i < len(con_hints); i++ {
    	hints := strings.Split(con_hints[i], ":")
    	b.hints[i] = make([][]int, len(hints))
		for h:= 0; h < len(hints); h++ {
			hints_spl := strings.Split(hints[h], "-")
			b.hints[i][h] = make([]int, len(hints_spl))
			for hs := 0; hs < len(hints_spl); hs++ {
				r, err := strconv.ParseInt(hints_spl[hs], 10, 0)
				if err != nil { log.Fatal(err) }
				b.hints[i][h][hs] = int(r)
			}
		}
    }
    
	// calculating longest hint row and longest hint column 
    tmp := 0
    tmp2 := 0
    for i := 0; i < len(b.hints[0]); i++ {
    	for j := 0; j < len(b.hints[0][i]); j++ {
    		tmp += len(string(b.hints[0][i][j]))
    		tmp += 1
    		tmp2 += 1
    	}
    	if tmp > b.lhr {
    		b.lhr = tmp
    	}
    	if tmp2 > b.mhr {
    		b.mhr = tmp2
    	}
    	tmp = 0
    	tmp2 = 0
    }
    for i := 0; i < len(b.hints[1]); i++ {
    	for j := 0; j < len(b.hints[1][i]); j++ {
    		tmp += len(string(b.hints[1][i][j]))
    		tmp += 1
    		tmp2 += 1
    	}
    	if tmp > b.lhc {
    		b.lhc = tmp
    	}
    	if tmp2 > b.mhc {
    		b.mhc = tmp2
    	}
    	tmp = 0
    	tmp2 = 0
    }
}
func (b *NonoviBoard) blankBoard(h int, w int) {
	b.board = make([][]int, h)
	for y:= 0; y < h; y++ {
		b.board[y] = make([]int, w)
	    for x:= 0; x < w; x++ {
	    	b.board[y][x] = 0
	    }
	} 
}

func (b *NonoviBoard) drawBoard() {
	// drawing hints columns
	for r := b.mhc; r > 0; r-- {
		fmt.Print(strings.Repeat(" ", b.lhr), "| ")
		for c := 0; c < len(b.hints[1]); c++ {
			if len(b.hints[1][c]) >= r {
				if cursor[0] == c {
					fmt.Print(colb_yellow, col_black, b.hints[1][c][len(b.hints[1][c]) - r], col_reset, " ")
				} else {
					fmt.Print(col_yellow, b.hints[1][c][len(b.hints[1][c]) - r], col_reset, " ")
				}
			} else {
				fmt.Print("  ")
			}
		}
		fmt.Print("\n")
	}
	// drawing the divider between hints for columns and hints for rows + board
	fmt.Println(strings.Repeat("-", b.lhr + 3 + len(b.board[0]) * 2))

	// drawing hints for rows and board
    for y := 0; y < len(b.board); y++ {
    	fmt.Print(strings.Repeat(" ", b.lhr - len(b.hints[0][y]) * 2))
    	for h := len(b.hints[0][y]); h >= 0; h-- {
    		if len(b.hints[0][y]) > h {
    			if cursor[1] == y {
    				fmt.Print(colb_yellow, col_black, b.hints[0][y][len(b.hints[0][y]) - h - 1], col_reset, " ")
    			} else {
    				fmt.Print(col_yellow, b.hints[0][y][len(b.hints[0][y]) - h - 1], col_reset, " ")
    			}
    		} 
    	}
    	fmt.Print("| ")
        for x := 0; x < len(b.board[y]); x++ {
        	if cursor[0] == x && cursor[1] == y && win != 1 {
        		fmt.Print(colb_red, b.board[y][x], col_reset, " ")
        	} else if b.board[y][x] == 1 { 
        		fmt.Print(colb_cyan, b.board[y][x], col_reset,  " ")
        	} else  {
        		// green marks on zeros in selected row & column
        		/*if cursor[0] == x || cursor[1] == y {
        			fmt.Print(colb_green, b.board[y][x], col_reset, " ")
        		} else {
        			fmt.Print(b.board[y][x], " ")
        		}*/
        		fmt.Print(b.board[y][x], " ")
        	}
        }
        fmt.Print("\n")
    }
}

func (b *NonoviBoard) drawEditor() {
	fmt.Println(strings.Repeat("-", 2 + len(b.board[0]) * 2))
	for y:= 0; y < len(b.board); y++ {
		fmt.Print("| ")

		for x := 0; x < len(b.board[y]); x++ {
			if cursor[0] == x && cursor[1] == y && win != 1 {
				fmt.Print(colb_red, b.board[y][x], col_reset, " ")
			} else if b.board[y][x] == 1 { 
		    	fmt.Print(colb_cyan, b.board[y][x], col_reset,  " ")
			} else {
		    	fmt.Print(b.board[y][x], " ")
			}
		}
		fmt.Print("\n")
	}
}

func game(fp string) {
	// loading final board
	var fb NonoviBoard 
	fb.loadBoard(userhome + config.NnvsDirectory + fp + ".nnv")
	if len(fb.board) == 0 {
		fmt.Println("Unable to open this fp; are you sure it exists?")
		return
	}
	if len(fb.board[0]) != len(fb.hints[1]) || len(fb.board) != len(fb.hints[0]) {
		fmt.Println("The file seems to be not formatted correctly; are you sure it's a valid nnv file?")
		return
	}
	// loading blank player board
	var bb NonoviBoard
	bb.blankBoard(len(fb.board), len(fb.board[0]))
	// copying final board metadata to blank player board
	bb.hints = fb.hints
	bb.lhc = fb.lhc
	bb.lhr = fb.lhr
	bb.mhc = fb.mhc
	bb.mhr = fb.mhr
		
	fmt.Print("\033[H\033[2J")
	bb.drawBoard()
	for {
		// board redraw
		if redraw == 1 {
			fmt.Print("\033[H\033[2J")
			bb.drawBoard()
			redraw = 0
		}
		// end game check
		if win == 1 {
			fmt.Println("You have won, congratulations!")
			win = 0
			cursor[0] = 0
			cursor[1] = 0
			return
		}
		// getting input and actions	
		char, key, err := keyboard.GetSingleKey()
		if err != nil { log.Fatal(err) }
		switch char {
			case 100: // d
				if cursor[0] + 1 < len(bb.board[0]) {
					cursor[0] += 1
					redraw = 1
				} 	
			case 97: // a
				if cursor[0] - 1 >= 0 {
					cursor[0] -= 1
					redraw = 1
				} 
			case 115: // s
				if cursor[1] + 1 < len(bb.board) {
					cursor[1] += 1
					redraw = 1
				} 
			case 119: // w
				if cursor[1] - 1 >= 0 {
					cursor[1] -= 1
					redraw = 1
				} 
			case 120:
				bb.board[cursor[1]][cursor[0]] = 1
				redraw = 1
			case 99:
				bb.board[cursor[1]][cursor[0]] = 0
				redraw = 1
			case 113:
				os.Exit(0)
		}
		switch key {
			case 65514: // arrow right
				if cursor[0] + 1 < len(bb.board[0]) {
					cursor[0] += 1
					redraw = 1
				} 
			case 65515: // arrow left
				if cursor[0] - 1 >= 0 {
					cursor[0] -= 1
					redraw = 1
				} 
			case 65516: // arrow down
				if cursor[1] + 1 < len(bb.board) {
					cursor[1] += 1
					redraw = 1
				} 
			case 65517: // arrow up
				if cursor[1] - 1 >= 0 {
					cursor[1] -= 1
					redraw = 1
				} 
			case 32: // space
				if bb.board[cursor[1]][cursor[0]] == 0 {
					bb.board[cursor[1]][cursor[0]] = 1
				} else {
					bb.board[cursor[1]][cursor[0]] = 0
				}
				redraw = 1
			case 3:
				os.Exit(0)
		}
		// win condition check
		if reflect.DeepEqual(bb.board, fb.board) == true { win = 1 }
	}
}

func editor(h int, w int) {
	var bb NonoviBoard
	bb.blankBoard(h, w)

	fmt.Print("\033[H\033[2J")
	bb.drawEditor()
	for {
		if redraw == 1 {
			fmt.Print("\033[H\033[2J")
			bb.drawEditor()
			redraw = 0
		}
		if finish == 1 {
			fmt.Println("Where to save your file?")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			in := scanner.Text()
			f, err := os.Create(userhome + config.NnvsDirectory + in + ".nnv")
			defer f.Close()
			if err != nil { log.Fatal(err) }
			wrt := bufio.NewWriter(f)
			_, err = wrt.WriteString("b-\n")
			if err != nil { log.Fatal(err) }
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if bb.board[y][x] == 1 {
						_, err := wrt.WriteString("x")
						if err != nil { log.Fatal(err) }
					} else if bb.board[y][x] == 0 {
						_, err := wrt.WriteString("_")
						if err != nil { log.Fatal(err) }
					}
					if err != nil { log.Fatal(err) }
				}
				wrt.WriteString("\n")
			}
			_, err = wrt.WriteString("eb-\n")
			if err != nil { log.Fatal(err) }

			bb.hints = make([][][]int, 2)
			bb.hints[0] = make([][]int, h)
			bb.hints[0][0] = make([]int, 0)
			bb.hints[1] = make([][]int, w)
			bb.hints[1][0] = make([]int, 0)
		
			tmp := 0
			for y := 0; y < w; y++ {
				for x := 0; x < h; x++ {
					if bb.board[x][y] == 1 {
						tmp += 1
					} else if bb.board[x][y] == 0 {
						if tmp != 0 {
							bb.hints[1][y] = append(bb.hints[1][y], tmp)
							tmp = 0
						}
					}
				}
				if tmp != 0 {
					bb.hints[1][y] = append(bb.hints[1][y], tmp)
					tmp = 0
				}
				if len(bb.hints[1][y]) == 0 { bb.hints[1][y] = append(bb.hints[1][y], 0) }
			}
			tmp = 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if bb.board[y][x] == 1 {
						tmp += 1
					} else if bb.board[y][x] == 0 {
						if tmp != 0 {
							bb.hints[0][y] = append(bb.hints[0][y], tmp)
							tmp = 0
						}
					}
				}
				if tmp != 0 {
					bb.hints[0][y] = append(bb.hints[0][y], tmp)
					tmp = 0
				}
				if len(bb.hints[0][y]) == 0 { bb.hints[0][y] = append(bb.hints[0][y], 0) }
			}
			_, err = wrt.WriteString("h-\n")
			if err != nil { log.Fatal(err) }
			for x := 0; x < h; x++ {
				for i := 0; i < len(bb.hints[0][x]); i++ {
					str := strconv.Itoa(bb.hints[0][x][i])
					wrt.WriteString(str)
					if i + 1 < len(bb.hints[0][x]) { wrt.WriteString("-") }
				}
				if x + 1 < h { wrt.WriteString(":") }
			}
			
			wrt.WriteString("\n")
			for y:= 0; y < w; y++ {
				for i := 0; i < len(bb.hints[1][y]); i++ {
					str := strconv.Itoa(bb.hints[1][y][i])
					wrt.WriteString(str)
					if i + 1 < len(bb.hints[1][y]) { wrt.WriteString("-") }
				}
				if y + 1 < w { wrt.WriteString(":") }
			}
			wrt.WriteString("\n")
			_, err = wrt.WriteString("eh-\n")
			if err != nil { log.Fatal(err) }
			
			wrt.Flush()		
			finish = 0
			cursor[0] = 0
			cursor[1] = 0
			return
		}
		char, key, err := keyboard.GetSingleKey()
		if err != nil { log.Fatal(err) }
		switch char {
			case 100: // d
				if cursor[0] + 1 < len(bb.board[0]) {
					cursor[0] += 1
					redraw = 1
				} 
			case 97: // a
				if cursor[0] - 1 >= 0 {
					cursor[0] -= 1
					redraw = 1
				}
			case 115: // s
				if cursor[1] + 1 < len(bb.board) {
					cursor[1] += 1
					redraw = 1
				} 
			case 119: // w
				if cursor[1] - 1 >= 0 {
					cursor[1] -= 1
					redraw = 1
				}
			case 120:
				bb.board[cursor[1]][cursor[0]] = 1
				redraw = 1
			case 99:
				bb.board[cursor[1]][cursor[0]] = 0
				redraw = 1
			case 113:
				os.Exit(0)
			case 101:
				finish = 1
		}
		switch key {
			case 65514: // arrow right
				if cursor[0] + 1 < len(bb.board[0]) {
					cursor[0] += 1
					redraw = 1
				}	
			case 65515: //arrow left
				if cursor[0] - 1 >= 0 {
					cursor[0] -= 1
					redraw = 1
				}				
			case 65516: // arrow down
				if cursor[1] + 1 < len(bb.board) {
					cursor[1] += 1
					redraw = 1
				}
			case 65517: //arrow up
				if cursor[1] - 1 >= 0 {
					cursor[1] -= 1
					redraw = 1
				} 
			case 32: // space
				if bb.board[cursor[1]][cursor[0]] == 0 {
					bb.board[cursor[1]][cursor[0]] = 1
				} else {
					bb.board[cursor[1]][cursor[0]] = 0
				}
				redraw = 1	
			case 3:
				os.Exit(0)
		}
		fmt.Println(key, char)
	}
}

func listLocal() {
	files, err := readDirectory(userhome + config.NnvsDirectory)
	if err != nil { fmt.Println("Error parsing local files"); return }
	for _, file := range files {
		name := strings.Split(file, ".")[0]
		fmt.Println(col_cyan, " * ", col_reset, name)
	}
}

func listmd5Local() {
	files, err := readDirectory(userhome + config.NnvsDirectory)
	if err != nil { fmt.Println("Error parsing local files"); return }
	for _, file := range files {
		name := strings.Split(file, ".")[0]
		content, err := os.ReadFile(userhome + config.NnvsDirectory + name + ".nnv")
		if err != nil { fmt.Println("Error parsing local files"); return }		
		var hash = md5.Sum(content)
		hash_str := base64.StdEncoding.EncodeToString(hash[:])
		fmt.Println(col_cyan, " * ", col_reset, name, hash_str)
	}
}

func listRemote() {
	resp, err := http.Get(config.Remote + "/get_nnvs")
	if err != nil { fmt.Println("Error fetching remote"); return }
	body, err := io.ReadAll(resp.Body)
	if err != nil { fmt.Println("Error parsing remote"); return }
	var l nnvsResponse
	err = json.Unmarshal([]byte(body), &l)
	if err != nil { fmt.Println("Error parsing remote"); return }
	for i := 0; i < len(l.Nnvs); i++ {
		status := 0
		content, err := os.ReadFile(userhome + config.NnvsDirectory + l.Nnvs[i][0] + ".nnv")
		if err != nil { 
			status = 0
		} else {
			var hash = md5.Sum(content)
			hash_str := base64.StdEncoding.EncodeToString(hash[:])
			if hash_str == l.Nnvs[i][1] {
				status = 2
			} else if hash_str != l.Nnvs[i][1] {
				status = 1
			}
			
		}
		switch status {
			case 2:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_green, "✓", col_reset)
			case 1:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_yellow, "?", col_reset)
			default:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_red, "×", col_reset)		
		}
	}
}

func listmd5Remote() {
	resp, err := http.Get(config.Remote + "/get_nnvs")
	if err != nil { fmt.Println("Error fetching remote"); return }
	body, err := io.ReadAll(resp.Body)
	if err != nil { fmt.Println("Error parsing remote"); return }
	var l nnvsResponse
	err = json.Unmarshal([]byte(body), &l)
	if err != nil { fmt.Println("Error parsing remote"); return }
	for i := 0; i < len(l.Nnvs); i++ {
		status := 0
		hash_str := ""
		content, err := os.ReadFile(userhome + config.NnvsDirectory + l.Nnvs[i][0] + ".nnv")
		if err != nil { 
			status = 0
		} else {
			var hash = md5.Sum(content)
			hash_str = base64.StdEncoding.EncodeToString(hash[:])
			if hash_str == l.Nnvs[i][1] {
				status = 2
			} else if hash_str != l.Nnvs[i][1] {
				status = 1
			}
			
		}
		switch status {
			case 2:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_green, "✓", col_reset, hash_str)
			case 1:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_yellow, "?", col_reset, hash_str)
			default:
				fmt.Println(col_cyan, " * ", col_reset, l.Nnvs[i][0], col_red, "×", col_reset, hash_str)
		}
	}
}

func getRemote(id string) {
	resp, err := http.Get(config.Remote + "/get_nnv/" + id)
	if err != nil { fmt.Println("Error fetching remote"); return }
	body, err := io.ReadAll(resp.Body)
	if err != nil { fmt.Println("Error parsing remote"); return }
	var r nnvResponse
	err = json.Unmarshal([]byte(body), &r)
	if err != nil { fmt.Println("Error parsing remote"); return }
	nnv := Nnv{id, r.Nnv[0], r.Nnv[1]}

	f, err := os.Create(userhome + "/.config/nonovi/nnvs/" + id + ".nnv")
	defer f.Close()
	if err != nil { log.Fatal(err) }
	wrt := bufio.NewWriter(f)
	_, err = wrt.WriteString(nnv.Board)
	if err != nil { log.Fatal(err) }
	wrt.Flush()	
}

func main() {
	makeNnvsDirectory()
	// if a file was provided as arg, play a game
	if len(args) == 1 {
			game(args[0])
	}
	fmt.Println("Welcome to Nonovi!\nType 'help' for help.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		scanner.Scan()
		in := scanner.Text()
		in_spl := strings.Split(in, " ")
		switch in_spl[0] {
			case "play":
				game(in_spl[1])
				continue
			case "editor":
				if len(in_spl) != 3 { fmt.Println("Incorrect numbers of args passed"); continue }
				h, err := strconv.ParseInt(in_spl[1], 10, 0)
				if err != nil { fmt.Println("Error parsing height"); continue }
				w, err := strconv.ParseInt(in_spl[2], 10, 0)
				if err != nil { fmt.Println("Error parsing width"); continue }
				editor(int(h), int(w))
				continue
			case "list":
				switch len(in_spl) {
					case 2:
						switch in_spl[1] {
							case "remote":
								listRemote()
							case "local":
								listLocal()
							default:
								fmt.Println("Invalid list option. Defaulting to local")
								listLocal()
						}
					case 3:
						switch in_spl[1] {
							case "remote":
								switch in_spl[2] {
									case "md5":
										listmd5Remote()
									case "local":
										listRemote()
									default:
										fmt.Println("Invalid list option. Defaulting to local")
										listLocal()
								}
							case "local":
								switch in_spl[2] {
									case "md5":
										listmd5Local()
									case "local":
										listLocal()
									default:
										fmt.Println("Invalid list option. Defaulting to local")
										listLocal()
								}
						}
					default:
						listLocal()
					
				}
			case "get":
				if len(in_spl) != 2 { fmt.Println("Incorrect numbers of args passed"); continue}
				getRemote(in_spl[1])
				continue
			case "help":
				fmt.Println( col_cyan + "\n Nonovi v1.1\n" + col_reset,"(c) 2022 Maciej Bromirski\n")
				fmt.Println(col_yellow + " play" + col_reset, "< title >\n", "To play the selected by title nonogram", "\n -----")
				fmt.Println(col_yellow + " list" + col_reset, "[ remote ]\n", "To list available nonograms. With optional remote will list all nonograms available on remote")
				fmt.Println(" If listing remote, files are marked with icons to reflect their state on local")
				fmt.Println(col_green + " ✓" + col_reset, "File present")
				fmt.Println(col_yellow + " ?" + col_reset, "File present but modified")
				fmt.Println(col_red + " ×" + col_reset, "File not present", "\n -----")
				fmt.Println(col_yellow + " editor" + col_reset, "< heigth > < width >\n", "To open the editor with selected heigth and width", "\n -----")
				fmt.Println(col_yellow + " help" + col_reset, "\n", "To display this message", "\n -----")
				continue
			case "quit":
				os.Exit(0)
			default:
				fmt.Println("Invalid command. Use \"help\" for help")			
		}
	}
}
