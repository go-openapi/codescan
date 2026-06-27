---
title: "swagger:file"
weight: 60
description: "Marks a parameter or response body as a binary file (`{type: file}`)."
---


## What it does

Marks a parameter or response body as a binary file
(`{type: file}`). The scanner emits the file-type marker without
further introspection of the Go type.

## Where it goes

On a struct field doc inside a `swagger:parameters` (multipart file
upload) or `swagger:response` (file download) struct.

## Syntax

```ebnf
FileBlock = ANN_FILE , [ Title ] , [ Description ] ;
```

Takes no argument — an optional title/description may follow on the
doc comment.

## Supported keywords

Standard parameter / response keywords; the file marker stacks with
`in:` and other parameter shape keywords. See the
[keywords reference]({{% relref "keywords" %}}).

## Example

```go
// UploadParams declares a multipart file upload.
//
// swagger:parameters uploadFile
type UploadParams struct {
	// File is the uploaded asset.
	//
	// in: formData
	// swagger:file
	File io.ReadCloser `json:"file"`
}
```
