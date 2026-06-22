// Package art holds the embedded terminal artwork shown across the UI, such as
// the rotating braille globe in the main menu header. The frames are generated
// offline (see the DO NOT EDIT banner in each generated file) from public-domain
// geographic data so the binary stays self-contained and pure Go.
package art

//go:generate go run ../../tools/genglobe -o globe.go
