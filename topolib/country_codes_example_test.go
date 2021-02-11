package topolib_test

import (
	"fmt"

	"github.com/9seconds/topographer/topolib"
)

func ExampleNormalizeAlpha2Code() {
	fmt.Println(topolib.NormalizeAlpha2Code("ru"))
	// output: RU
}

func ExampleNormalizeAlpha2Code_yugoslavia() {
	fmt.Println(topolib.NormalizeAlpha2Code("YU"))
	// output: CS
}

func ExampleAlpha2ToCountryCode() {
	code := topolib.Alpha2ToCountryCode("uk")

	fmt.Println(code.String())
	fmt.Println(code.Details().Name.BaseLang.Common)
	// output:
	// GB
	// United Kingdom
}

func ExampleAlpha3ToCountryCode() {
	code := topolib.Alpha3ToCountryCode("ita")

	fmt.Println(code.String())
	fmt.Println(code.Details().Name.BaseLang.Common)
	// output:
	// IT
	// Italy
}
