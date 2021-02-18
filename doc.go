// Topographer is a service to resolve geolocation data for given set of
// IP addresses.
//
// Idea is simple: you have an IP address like 1.2.3.4. And you want to
// know where this user comes from, which city. So, this is a geoloction
// task.
//
// Tool itself is organized into 3 logical parts:
//
// Topolib
//
// topolib is a main package of the application which contains
// Topographer struct and main logic related to geolocation. Topographer
// has a set of pluggable providers and some options, mostly optional.
// It has its own API and can act as http.Handler.
//
// Providers
//
// This package has a set of provider implemntations which cover most
// of the usecases. If you need MaxMind, it is there, no need to do
// anything else.
//
// Topographer
//
// A main package itself is an example of how to wire both topolib and
// providers. But this is a full example which providers CLI. Resulting
// binary starts http server and you can use it in your infrastructure
// as is.
package main
