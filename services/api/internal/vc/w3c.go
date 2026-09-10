package vc

import (
	"time"
)

// W3C Context URIs
const (
	W3CCredentialsContextV1 = "https://www.w3.org/2018/credentials/v1"
	W3CDIDContextV1         = "https://www.w3.org/ns/did/v1"
	W3CSecurityJWS2020      = "https://w3id.org/security/suites/jws-2020/v1"
	EkaIDContextV1          = "https://id.eka.dev/contexts/v1"
)

// Standard Credential Types
const (
	TypeVerifiableCredential     = "VerifiableCredential"
	TypeEkaIdentityCredential    = "EkaIdentityCredential"
	TypeEkaVerifiableAttestation = "EkaVerifiableAttestation"
	TypeVerifiablePresentation   = "VerifiablePresentation"
)

// VerifiableCredential implements the W3C Verifiable Credentials Data Model 1.1 / 2.0
type VerifiableCredential struct {
	Context           []string          `json:"@context"`
	ID                string            `json:"id"`
	Type              []string          `json:"type"`
	Issuer            Issuer            `json:"issuer"`
	IssuanceDate      time.Time         `json:"issuanceDate"`
	ExpirationDate    *time.Time        `json:"expirationDate,omitempty"`
	CredentialSubject CredentialSubject `json:"credentialSubject"`
	Proof             *Proof            `json:"proof,omitempty"`
}

// Issuer identifies the issuing authority using a DID
type Issuer struct {
	ID   string `json:"id"`             // e.g. "did:eka:org:acme-tech" or "did:eka:issuer:platform"
	Name string `json:"name,omitempty"` // e.g. "Acme Technologies Ltd."
}

// CredentialSubject contains the claims asserted about the identity subject
type CredentialSubject struct {
	ID     string                 `json:"id"` // e.g. "did:eka:EKA-7K4M-92PX"
	Claims map[string]interface{} `json:"claims,omitempty"`
}

// Proof contains the cryptographic proof securing the credential against tampering
type Proof struct {
	Type               string    `json:"type"`               // e.g. "JsonWebSignature2020" / "EkaSignature2026"
	Created            time.Time `json:"created"`
	VerificationMethod string    `json:"verificationMethod"` // e.g. "did:eka:EKA-7K4M-92PX#key-1"
	ProofPurpose       string    `json:"proofPurpose"`       // "assertionMethod"
	Jws                string    `json:"jws,omitempty"`
	ProofValue         string    `json:"proofValue,omitempty"`
}

// VerifiablePresentation packages one or more credentials for presentation to a verifier
type VerifiablePresentation struct {
	Context              []string               `json:"@context"`
	ID                   string                 `json:"id"`
	Type                 []string               `json:"type"`
	VerifiableCredential []VerifiableCredential `json:"verifiableCredential"`
	Holder               string                 `json:"holder,omitempty"` // e.g. "did:eka:EKA-7K4M-92PX"
	Proof                *Proof                 `json:"proof,omitempty"`
}

// DIDDocument implements W3C Decentralized Identifiers (DIDs) v1.0
type DIDDocument struct {
	Context            []string             `json:"@context"`
	ID                 string               `json:"id"` // "did:eka:EKA-7K4M-92PX"
	AlsoKnownAs        []string             `json:"alsoKnownAs,omitempty"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []string             `json:"authentication"`
	AssertionMethod    []string             `json:"assertionMethod"`
	Service            []DIDService         `json:"service,omitempty"`
}

// VerificationMethod defines cryptographic keys in a DID Document
type VerificationMethod struct {
	ID           string                 `json:"id"` // "did:eka:EKA-7K4M-92PX#key-1"
	Type         string                 `json:"type"`
	Controller   string                 `json:"controller"` // "did:eka:EKA-7K4M-92PX"
	PublicKeyJwk map[string]interface{} `json:"publicKeyJwk,omitempty"`
}

// DIDService defines service endpoints in a DID Document
type DIDService struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}
