# GO REST Framework

## Contents

- Request Generator
- Reverse Proxy
- REST API

## Overview

This project will is used to protoype and do performance analysis on how best to manage requests.

## Implementations

1. 1 request per reverse proxy and 1 request per backend

   - Shut down once we respond to 1 request

2. many requests per reverse proxy and 1 request per backend

   - leave reverse proxy up until idle for x time
   - shut down backend after responding to 1 request

3. many requests per rever proxy and many requests per backend

   - leave reverse proxy up until idle for x time
   - leave backend up until idle for x time

### What does x time mean?

- expenential growth per request up to a cap?
- for idle tests, we can just leave it up permenantly
