package main

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	mBlockSize = int64(1024 * 1024)
)

func main() {
	app := cli.NewApp()

	app.CustomAppHelpTemplate =
		`
NAME:
	{{.Name}} - {{.Usage}}
USAGE:
	{{.HelpName}} -i <.xva file> -o <.raw output file>
OPTIONS:
	{{range .VisibleFlags}}{{.}}
	{{end}}
`
	app.Flags = []cli.Flag{
		&cli.PathFlag{
			Name:     "input",
			Aliases:  []string{"i"},
			Required: true,
		},
		&cli.PathFlag{
			Name:     "output",
			Aliases:  []string{"o"},
			Required: true,
		},
	}

	app.Action = func(c *cli.Context) error {
		zeroBlock := make([]byte, mBlockSize)

		xvaFile, err := os.Open(c.Path("input"))

		if err != nil {
			return err
		}

		xvaReader := tar.NewReader(xvaFile)

		rawFile, err := openOrCreatefile(c.Path("output"))

		if err != nil {
			return err
		}

		i := 0

		for {
			h, err := xvaReader.Next()

			if err != nil && err != io.EOF {
				return err
			} else if err == io.EOF {
				return nil
			}

			if !strings.Contains(h.Name, "Ref") {
				continue
			}

			if strings.Contains(h.Name, "xxhash") { //TODO : REMOVER
				continue
			}

			nBlock, err := strconv.Atoi(strings.Split(h.Name, "/")[1])

			if err != nil {
				return err
			}

			for i < nBlock {
				if err = writeFile(rawFile, int64(i), zeroBlock); err != nil {
					return err
				}

				i++
				log.Println(i, "MB")
			}

			bs, err := ioutil.ReadAll(xvaReader)

			if err = writeFile(rawFile, int64(i), bs); err != nil {
				return err
			}

			i++
			log.Println(i, "MB")
		}
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func writeFile(output *os.File, pos int64, bs []byte) (err error) {
	if bs == nil {
		for i := int64(0); i < mBlockSize; i++ {
			_, err = output.WriteAt([]byte{0x0}, (mBlockSize*pos)+i)

			if err != nil {
				return err
			}
		}
	} else {
		_, err = output.WriteAt(bs, (mBlockSize * pos))

		if err != nil {
			return err
		}
	}

	return nil
}

func openOrCreatefile(p string) (f *os.File, err error) {
	if _, err = os.Stat(p); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(p), 0755); err != nil {
			return nil, err
		}
		if f, err = os.Create(p); err != nil {
			return nil, err
		}
	} else {
		if f, err = os.OpenFile(p, os.O_WRONLY, os.ModePerm); err != nil {
			return nil, err
		}
	}

	return f, err
}
