package main

import "github.com/blackjack/webcam"
import (
	gc "github.com/rthornton128/goncurses"
)
import "os"
import "fmt"
import "net"
import "encoding/binary"
import "time"
import "math/rand"

func InitTermColours() {
	gc.StartColor()
	gc.UseDefaultColors()

	NColors := 6 /* 6 * 6 * 6 = 216 colours */
	ColNum := 0
	for r := 0; r < NColors; r++ {
		for g := 0; g < NColors; g++ {
			for b := 0; b < NColors; b++ {
				gc.InitColor(int16(ColNum), int16(r*200), int16(g*200), int16(b*200))
				gc.InitPair(int16(ColNum)+1, int16(ColNum), int16(ColNum)) //  gc.C_BLACK)
				ColNum += 1

			}
		}
	}
}

func RGBToColnum(r, g, b int) int16 {
	return int16(4*b/200 + 6*(4*g/200) + 6*6*(4*r/200) + 1)
}

func YUYVToRGB(y, u, v int) (int, int, int) {
	y2 := y
	u2 := u - 128
	v2 := v - 128

	var r int = y2 + ((v2 * 37221) >> 15)
	var g int = y2 - (((u2 * 12975) + (v2 * 18949)) >> 15)
	var b int = y2 + ((u2 * 66883) >> 15)

	if r < 0 {
		r = 0
	} else if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	} else if g > 255 {
		g = 255
	}

	if b < 0 {
		b = 0
	} else if b > 255 {
		b = 255
	}

	return r, g, b
}
func ReadFrameFromCamera(cam *webcam.Webcam, Buffer *[100][35]byte) {
	timeout := uint32(5) //5 seconds
	err := cam.WaitForFrame(timeout)
	switch err.(type) {
	case nil:
	case *webcam.Timeout:
		fmt.Fprint(os.Stderr, err.Error())
		return
		//continue
	default:
		panic(err.Error())
	}
	frame, err := cam.ReadFrame()
	if len(frame) != 0 {
		width := 320
		height := 240
		skip := 1
		for j := 0; j < width*height*2; j = j + 4*skip {
			y := frame[j]
			u := frame[j+1]
			v := frame[j+3]
			r, g, b := YUYVToRGB(int(y), int(u), int(v))
			far := j / 2
			x := far % 320
			yv := far / 320
			xp := 100 * x / 320
			yp := 35 * yv / 240
			Buffer[xp][yp] = byte(RGBToColnum(r, g, b))
		}

	} else if err != nil {
		panic(err.Error())
	}
}
func RenderBuffer(stdscr *gc.Window, Buffer *[100][35]byte) {

	stdscr.Refresh()
	for w := 0; w < 100; w++ {
		for h := 0; h < 35; h++ {
			c := int16(Buffer[w][h])
			stdscr.ColorOn(c)
			//stdscr.AttrOn(gc.A_BOLD | gc.ColorPair(c))
			stdscr.MoveAddChar(h, w, '#')
			//stdscr.AttrOff(gc.A_BOLD | gc.ColorPair(c))
			stdscr.ColorOff(c)

		}
	}
}
func SendBufferToServer(conn net.Conn, Buffer [100][35]byte) {
	binary.Write(conn, binary.LittleEndian, Buffer)
}
func ReadBufferFromServer(conn net.Conn, Buffer *[100][35]byte) {
	binary.Read(conn, binary.LittleEndian, Buffer)
}
func InitCamera() *webcam.Webcam {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		panic(err.Error())
	}

	var format webcam.PixelFormat = 1448695129 /*V4L2_PIX_FMT_YUYV YUV 4:2:2 */
	size := webcam.FrameSize{320, 320, 0, 240, 240, 0}

	_, _, _, err = cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		panic(err.Error())
	}
	err = cam.StartStreaming()
	if err != nil {
		panic(err.Error())
	}
	return cam
}
func RandomizeBuffer(Buffer *[100][35]byte) {
	for w := 0; w < 100; w++ {
		for h := 0; h < 35; h++ {
			Buffer[w][h] = byte(rand.Intn(216))
		}
	}
}

func main() {
	var ReadLocalCamera bool
	var NetworkMode bool
	var ServerAddr string
	var MyNick string
	var TheirNick string
	var cam *webcam.Webcam
	args := os.Args
	if len(args) == 1 {
		ReadLocalCamera = true
		NetworkMode = false
	} else if len(args) == 5 {
		if args[4] == "f" {
			ReadLocalCamera = true
		} else {
			ReadLocalCamera = false
		}
		NetworkMode = true
		ServerAddr = args[1]
		MyNick = args[2]
		TheirNick = args[3]
	} else {
		fmt.Printf("%s [server:port] [myname] [theirname] [fakevideo]", args[0])
		return
	}
	var conn net.Conn
	if NetworkMode {
		conn, _ = net.Dial("tcp", ServerAddr)

		fmt.Fprintf(conn, "%s\n", MyNick)    // Who I am
		fmt.Fprintf(conn, "%s\n", TheirNick) // Who I want to speak to
		fmt.Printf("Waiting for %s to connect ", TheirNick)
		fmt.Fscanf(conn, "Ready\n")
		//conn.SetReadDeadline(time.Now().Add(100*time.Millisecond))
	}

	stdscr, err := gc.Init()
	if err != nil {
		return
	}
	defer gc.End()

	InitTermColours()

	LocalScreen := [100][35]byte{}
	ToSend := [100][35]byte{}

	if ReadLocalCamera {
		cam = InitCamera()
		defer cam.Close()
	}

	for {

		if NetworkMode {
			if ReadLocalCamera {
				ReadFrameFromCamera(cam, &ToSend)
			} else {
				time.Sleep(100 * time.Millisecond)
				RandomizeBuffer(&ToSend)
			}
			SendBufferToServer(conn, ToSend)
			ReadBufferFromServer(conn, &LocalScreen)
			RenderBuffer(stdscr, &LocalScreen)
		} else { // Just display the camera image locally
			ReadFrameFromCamera(cam, &LocalScreen)
			RenderBuffer(stdscr, &LocalScreen)
		}

	}
}
