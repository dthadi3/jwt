/*
 * Copyright 2018-2020 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"
)

func TestImportValidation(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("imports should not generate an issue")
	}

	i.Type = Service
	vr = CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("imports should not generate an issue")
	}

	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(time.Hour).UTC().Unix()

	activation.ImportSubject = "test"
	activation.ImportType = Stream
	actJWT := encode(activation, ak2, t)

	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		t.Errorf("imports with token should be valid")
	}
}

func TestImportValidationExpiredToken(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}
	// test success, expiration is not checked
	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(-time.Hour).UTC().Unix()
	activation.ImportSubject = "test"
	activation.ImportType = Stream
	i.Token = encode(activation, ak2, t)
	vr := CreateValidationResults()
	i.Validate(akp, vr)
	if !vr.IsEmpty() {
		t.Errorf("Expired token should not trigger a validation issue")
	}
	// test failure, different issuer
	ak3 := createAccountNKey(t)
	activation = NewActivationClaims(akp)
	activation.Expires = time.Now().Add(-time.Hour).UTC().Unix()
	activation.ImportSubject = "test"
	activation.ImportType = Stream
	i.Token = encode(activation, ak3, t)
	vr = CreateValidationResults()
	i.Validate(akp, vr)
	if vr.IsEmpty() {
		t.Errorf("Issuer mismatch must trigger a validation issue")
	}
}

func TestImportValidationDifferentAccount(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	otherAccount := publicKey(createAccountNKey(t), t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}

	activation := NewActivationClaims(otherAccount)
	activation.Expires = time.Now().Add(-time.Hour).UTC().Unix()
	activation.ImportSubject = "test"
	activation.ImportType = Stream
	i.Token = encode(activation, ak2, t)
	vr := CreateValidationResults()
	i.Validate(akp, vr)
	if vr.IsEmpty() || !vr.IsBlocking(false) {
		t.Errorf("Expired import needs to result in a time check error")
	}
}

func TestImportValidationSigningKey(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	ak2Sk := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}

	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(time.Hour).UTC().Unix()
	activation.ImportSubject = "test"
	activation.ImportType = Stream
	activation.IssuerAccount = akp2
	i.Token = encode(activation, ak2Sk, t)
	vr := CreateValidationResults()
	i.Validate(akp, vr)
	if !vr.IsEmpty() {
		t.Errorf("Expired import needs to not result in an error")
	}
}

func TestInvalidImportType(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	i := &Import{Subject: "foo", Account: akp, To: "bar", Type: Unknown}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if vr.IsEmpty() {
		t.Errorf("imports without token or url should warn the caller")
	}

	if !vr.IsBlocking(true) {
		t.Errorf("invalid type is blocking")
	}
}

func TestInvalidImportToken(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	i := &Import{Subject: "foo", Account: akp, Token: "bad token", To: "bar", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if vr.IsEmpty() {
		t.Errorf("imports with a bad token or url should warn the caller")
	}

	if vr.IsBlocking(true) {
		t.Errorf("invalid type shouldnt be blocking")
	}
}

func TestInvalidImportURL(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	i := &Import{Subject: "foo", Account: akp, Token: "foo://bad token url", To: "bar", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if vr.IsEmpty() {
		t.Errorf("imports with a bad token or url should warn the caller")
	}

	if vr.IsBlocking(true) {
		t.Errorf("invalid type shouldnt be blocking")
	}
}

func TestInvalidImportTokenValuesValidation(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("imports should not generate an issue")
	}

	i.Type = Service
	vr = CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("imports should not generate an issue")
	}

	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(time.Hour).UTC().Unix()

	activation.ImportSubject = "test"
	activation.ImportType = Stream
	actJWT := encode(activation, ak2, t)

	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		t.Errorf("imports with token should be valid")
	}

	actJWT = encode(activation, ak, t) // wrong issuer
	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if vr.IsEmpty() {
		t.Errorf("imports with wrong issuer")
	}

	activation.Subject = akp2           // wrong subject
	actJWT = encode(activation, ak2, t) // right issuer
	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if vr.IsEmpty() {
		t.Errorf("imports with wrong issuer")
	}
}
func TestMissingAccountInImport(t *testing.T) {
	i := &Import{Subject: "foo", To: "bar", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if len(vr.Issues) != 1 {
		t.Errorf("expected only one issue")
	}

	if vr.IsBlocking(true) {
		t.Errorf("Missing Account is not blocking, must import failures are warnings")
	}
}

func TestServiceImportWithWildcard(t *testing.T) {
	i := &Import{Subject: "foo.*", Account: publicKey(createAccountNKey(t), t), To: "bar", Type: Service}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("expected no issue")
	}

	i.Subject = ">"
	vr = CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("expected no issue")
	}
}

func TestStreamImportWithWildcardPrefix(t *testing.T) {
	i := &Import{Subject: "foo", Account: publicKey(createAccountNKey(t), t), To: "bar.*", Type: Stream}

	vr := CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("expected no issue")
	}

	i.Subject = ">"
	vr = CreateValidationResults()
	i.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("expected no issue")
	}
}

func TestStreamImportInformationSharing(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	// broken import share won't work with streams
	i := &Import{Subject: "foo", Account: akp, Type: Stream, Share: true}
	vr := CreateValidationResults()
	i.Validate("", vr)

	if len(vr.Issues) != 1 {
		t.Errorf("should have registered 1 issues with this import, got %d", len(vr.Issues))
	}
	if !vr.IsBlocking(true) {
		t.Fatalf("issue is expected to be blocking")
	}
	// import share will work with service
	i.Type = Service
	vr = CreateValidationResults()
	i.Validate("", vr)

	if len(vr.Issues) != 0 {
		t.Errorf("should have registered 0 issues with this import, got %d", len(vr.Issues))
	}
}

func TestImportsValidation(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	i := &Import{Subject: "foo", Account: akp, To: "bar", Type: Stream}
	i2 := &Import{Subject: "foo.*", Account: akp, To: "bar", Type: Service}

	imports := &Imports{}
	imports.Add(i, i2)

	vr := CreateValidationResults()
	imports.Validate("", vr)

	if !vr.IsEmpty() {
		t.Errorf("no issues expected")
	}
}

func TestTokenURLImportValidation(t *testing.T) {
	ak := createAccountNKey(t)
	ak2 := createAccountNKey(t)
	akp := publicKey(ak, t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "test", Account: akp2, To: "bar", Type: Stream}

	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(time.Hour).UTC().Unix()
	activation.ImportSubject = "test"
	activation.ImportType = Stream

	actJWT := encode(activation, ak2, t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(actJWT))
	}))
	defer ts.Close()

	i.Token = ts.URL
	vr := CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		fmt.Printf("vr is %+v\n", vr)
		t.Errorf("imports with token url should be valid")
	}

	i.Token = "http://Bad URL"
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if vr.IsEmpty() {
		t.Errorf("imports with bad token url should be valid")
	}

	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bad jwt"))
	}))
	defer ts.Close()

	i.Token = ts.URL
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if vr.IsEmpty() {
		t.Errorf("imports with token url pointing to bad JWT")
	}

	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	i.Token = ts.URL
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if vr.IsEmpty() {
		t.Errorf("imports with token url pointing to bad url")
	}
}

func TestImportSubjectValidation(t *testing.T) {
	ak := createAccountNKey(t)
	akp := publicKey(ak, t)
	activation := NewActivationClaims(akp)
	activation.Expires = time.Now().Add(time.Hour).UTC().Unix()
	activation.ImportSubject = "one.*"
	activation.ImportType = Stream

	ak2 := createAccountNKey(t)
	akp2 := publicKey(ak2, t)
	i := &Import{Subject: "one.two", Account: akp2, To: "bar", Type: Stream}

	actJWT := encode(activation, ak2, t)
	i.Token = actJWT
	vr := CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		t.Log(vr.Issues[0].Description)
		t.Errorf("imports with valid contains subject should be valid")
	}

	activation.ImportSubject = "two"
	activation.ImportType = Stream
	actJWT = encode(activation, ak2, t)
	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		t.Errorf("imports with non-contains subject should be not valid")
	}

	activation.ImportSubject = ">"
	activation.ImportType = Stream
	actJWT = encode(activation, ak2, t)
	i.Token = actJWT
	vr = CreateValidationResults()
	i.Validate(akp, vr)

	if !vr.IsEmpty() {
		t.Errorf("imports with valid contains subject should be valid")
	}
}

func TestImportServiceDoubleToSubjectsValidation(t *testing.T) {
	akp := createAccountNKey(t)
	akp2 := createAccountNKey(t)
	apk := publicKey(akp, t)
	apk2 := publicKey(akp2, t)

	account := NewAccountClaims(apk)

	i := &Import{Subject: "one.two", Account: apk2, To: "foo.bar", Type: Service}
	account.Imports.Add(i)

	vr := CreateValidationResults()
	account.Validate(vr)

	if vr.IsBlocking(true) {
		t.Fatalf("Expected no blocking validation errors")
	}

	i2 := &Import{Subject: "two.three", Account: apk2, To: "foo.bar", Type: Service}
	account.Imports.Add(i2)

	vr = CreateValidationResults()
	account.Validate(vr)

	if !vr.IsBlocking(true) {
		t.Fatalf("Expected multiple import 'to' subjects to produce an error")
	}
}

func TestWildcard(t *testing.T) {
	account := NewAccountClaims(publicKey(createAccountNKey(t), t))

	i := &Import{Subject: ">", Account: publicKey(createAccountNKey(t), t), To: "foo.bar", Type: Service}
	account.Imports.Add(i)

	vr := CreateValidationResults()
	account.Validate(vr)

	if vr.IsBlocking(true) {
		t.Fatalf("Expected no blocking validation errors")
	}

}

func TestImport_Sorting(t *testing.T) {
	var imports Imports
	pk := publicKey(createAccountNKey(t), t)
	imports.Add(&Import{Subject: "x", Type: Service, Account: pk})
	imports.Add(&Import{Subject: "z", Type: Service, Account: pk})
	imports.Add(&Import{Subject: "y", Type: Service, Account: pk})
	if imports[0].Subject != "x" {
		t.Fatal("added import not in expected order")
	}
	sort.Sort(imports)
	if imports[0].Subject != "x" && imports[1].Subject != "y" && imports[2].Subject != "z" {
		t.Fatal("imports not sorted")
	}
}
