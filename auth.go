package marrow

import (
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthScheme string

const (
	BearerAuth = AuthScheme("Bearer")
	BasicAuth  = AuthScheme("Basic")
	TokenAuth  = AuthScheme("Token")
)

type AuthValue struct {
	Scheme AuthScheme
	Value  any
}

func Auth(scheme AuthScheme, value any) Resolvable {
	return AuthValue{
		Scheme: scheme,
		Value:  value,
	}
}

func (v AuthValue) ResolveValue(ctx Context) (av any, err error) {
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		var sv string
		var ok bool
		if sv, ok = rv.(string); !ok {
			sv = fmt.Sprintf("%v", rv)
		}
		if v.Scheme != "" {
			av = string(v.Scheme) + " " + sv
		} else {
			av = sv
		}
	}
	return av, err
}

func (v AuthValue) String() string {
	return fmt.Sprintf("Auth(%q, %v)", v.Scheme, v.Value)
}

type AuthUsernamePasswordValue struct {
	Username any
	Password any
}

func AuthUsernamePassword(username any, password any) Resolvable {
	return AuthUsernamePasswordValue{
		Username: username,
		Password: password,
	}
}

func (a AuthUsernamePasswordValue) ResolveValue(ctx Context) (av any, err error) {
	var uv any
	if uv, err = ResolveValue(a.Username, ctx); err == nil {
		var pv any
		if pv, err = ResolveValue(a.Password, ctx); err == nil {
			av = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", uv, pv)))
		}
	}
	return av, err
}

func (a AuthUsernamePasswordValue) String() string {
	return fmt.Sprintf("AuthUsernamePassword(%v, %v)", a.Username, a.Password)
}

type jwtValue struct {
	signingMethod jwt.SigningMethod
	secret        any
	claims        []ClaimValue
}

// Jwt creates an HS256-signed JWT with the given secret and claims
//
// this returns a resolvable value that can be used with Auth - which can then be used by Method_.AuthHeader
// example:
//
//	Method(GET, "do get").
//	    AuthHeader(BearerAuth, Jwt(
//	        Var("my-secret"),
//	        SubjectClaim(Var("my-user")),
//	        ExpireAfterClaim(5 * time.Minute))
func Jwt(secret any, claims ...ClaimValue) Resolvable {
	return jwtValue{
		signingMethod: jwt.SigningMethodHS256,
		secret:        secret,
		claims:        claims,
	}
}

// JwtHS384 creates an HS384-signed JWT with the given secret and claims
//
// this returns a resolvable value that can be used with Auth - which can then be used by Method_.AuthHeader
// example:
//
//	Method(GET, "do get").
//	    AuthHeader(BearerAuth, JwtHS384(
//	        Var("my-secret"),
//	        SubjectClaim(Var("my-user")),
//	        ExpireAfterClaim(5 * time.Minute))
func JwtHS384(secret any, claims ...ClaimValue) Resolvable {
	return jwtValue{
		signingMethod: jwt.SigningMethodHS384,
		secret:        secret,
		claims:        claims,
	}
}

// JwtHS512 creates an HS512-signed JWT with the given secret and claims
//
// this returns a resolvable value that can be used with Auth - which can then be used by Method_.AuthHeader
// example:
//
//	Method(GET, "do get").
//	    AuthHeader(BearerAuth, JwtHS512(
//	        Var("my-secret"),
//	        SubjectClaim(Var("my-user")),
//	        ExpireAfterClaim(5 * time.Minute))
func JwtHS512(secret any, claims ...ClaimValue) Resolvable {
	return jwtValue{
		signingMethod: jwt.SigningMethodHS512,
		secret:        secret,
		claims:        claims,
	}
}

func (j jwtValue) ResolveValue(ctx Context) (av any, err error) {
	var sv any
	if sv, err = ResolveValue(j.secret, ctx); err == nil {
		claims := make(jwt.MapClaims, len(j.claims))
		for i := 0; i < len(j.claims) && err == nil; i++ {
			if claim := j.claims[i]; claim != nil {
				var cv any
				if cv, err = ResolveValue(claim.Value(), ctx); err == nil {
					claims[claim.Name()] = cv
				}
			}
		}
		if err == nil {
			token := jwt.NewWithClaims(j.signingMethod, claims)
			switch svt := sv.(type) {
			case []byte:
				av, err = token.SignedString(svt)
			case string:
				av, err = token.SignedString([]byte(svt))
			default:
				av, err = token.SignedString([]byte(fmt.Sprintf("%v", sv)))
			}
		}
	}
	return av, err
}

type ClaimValue interface {
	Name() string
	Value() any
}

func Claim(name string, value any) ClaimValue {
	return claimValue{
		name:  name,
		value: value,
	}
}

func SubjectClaim(value any) ClaimValue {
	return claimValue{
		name:  "sub",
		value: value,
	}
}

func IssuerClaim(value any) ClaimValue {
	return claimValue{
		name:  "iss",
		value: value,
	}
}

func AudienceClaim(value any) ClaimValue {
	return claimValue{
		name:  "aud",
		value: value,
	}
}

func ExpireAtClaim(expiry time.Time) ClaimValue {
	return claimValue{
		name:  "exp",
		value: expiry.Unix(),
	}
}

func ExpireAfterClaim(dur time.Duration) ClaimValue {
	return claimValue{
		name: "exp",
		value: func() (any, error) {
			return time.Now().Add(dur).Unix(), nil
		},
	}
}

type claimValue struct {
	name  string
	value any
}

func (c claimValue) Name() string {
	return c.name
}

func (c claimValue) Value() any {
	return c.value
}
