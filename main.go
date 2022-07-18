package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const IapTokenHeader = "x-goog-iap-jwt-assertion"
const IapKeysURL = "https://www.gstatic.com/iap/verify/public_key-jwk"

// Flags
var listen string
var audience string
var acceptableSkew int
var verbose bool

var jwkCache *jwk.Cache
var parseOptions []jwt.ParseOption
var tokenEscaper = strings.NewReplacer("\n", "", "\r", "")

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address")
	flag.StringVar(&audience, "audience", "", "the JWT audience")
	flag.IntVar(&acceptableSkew, "skew", 0, "the acceptable skew in seconds")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
	flag.Parse()

	jwkCache = jwk.NewCache(context.Background())
	if err := jwkCache.Register(IapKeysURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		log.Fatalf("failed to register JWKs: %v", err)
	}

	if audience != "" {
		parseOptions = append(parseOptions, jwt.WithAudience(audience))
	} else {
		log.Fatalf("missing audience")
	}

	if acceptableSkew > 0 {
		parseOptions = append(parseOptions, jwt.WithAcceptableSkew(time.Duration(acceptableSkew)*time.Second))
	}

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		if err := validate(r); err != nil {
			log.Printf("failed to validate: %v", err)
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	log.Printf("listening on %s", listen)
	if err := http.ListenAndServe(listen, nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func validate(r *http.Request) (err error) {
	var token = r.Header.Get(IapTokenHeader)
	if token == "" {
		return fmt.Errorf("missing token from request")
	}

	if verbose {
		log.Printf("received token: %s", tokenEscaper.Replace(token))
	}

	for retry := 0; retry < 2; retry++ {
		var ks jwk.Set
		ks, err = jwkCache.Get(context.Background(), IapKeysURL)
		if err != nil {
			return fmt.Errorf("failed to get JWK keyset: %v", err)
		}

		var tok jwt.Token
		tok, err = jwt.ParseString(token, append(parseOptions, jwt.WithKeySet(ks))...)
		if err != nil {
			// Attempt to refresh the JWK keyset in case the keyset is stale and missing a key
			if retry == 0 {
				if _, err := jwkCache.Refresh(context.Background(), IapKeysURL); err != nil {
					return fmt.Errorf("failed to refresh JWK keyset: %v", err)
				}
				log.Printf("failed to parse token: %v, retrying", err)
			}
			continue
		}

		email, _ := tok.Get("email")
		log.Printf("successfully validated token for %s", email)
		break
	}

	return err
}
