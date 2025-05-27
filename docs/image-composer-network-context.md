# Image Composer Tool System Network Context

## Introduction

Image Composer Tool ![network diagram](assets/image-composer-network-diagram.drawio.svg) illustrates how different components of the product's system architecture communicate with each other. This type of diagram is useful for technical documentation, infrastructure planning, security review and troubleshooting.

### Network Security Considerations
Image Composer Tool downloads required packages using HTTP requests to the distribution specific package repos over TLS 1.2+ connections. Each of the package repos does server side validation on the package download requests. So it is expected that the system running the image composer tool is provisioned with a CA root chain.
 
 ### Package Sign Verification
 All the downloaded packages are integrity verified using GPG Public Keys and SHA256/MD5 checksum published at the package repos.