// This package provides a set of structs and functions which are used
// to geolocate given IP addresses.
//
// topolib is core of the topographer project. You can treat the rest of
// the application as an _example_ on how to use this library: how to
// pass parameters from HTTP requests, how to generate responses, how to
// implement providers.
//
// Topographer is a main entity of the topolib. This struct contains all
// logic related to IP geolocation: how to resolve IP adddresses, how to
// use worker pools, how to gather and track usage statistics.
//
// Topographer accepts ip address and returns ResolveResult: enriched
// and consolidated output of providers with chosen country/city.
package topolib
