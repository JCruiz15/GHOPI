FROM golang:1.19

# move to container path /app
WORKDIR /.

VOLUME /.config

# install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Declare environment variables
ENV GITHUB_CLIENTID=your-github-client-id
ENV GITHUB_SECRETID=your-github-secret
ENV OPENPROJECT_CLIENTID=your-openproject-client-id
ENV OPENPROJECT_SECRETID=your-openproject-secret
ENV PORT=8080
ENV URL_SUBPATH=
ENV API_KEY=myapikey

# copy project
COPY . .

# build project
RUN CGO_ENABLED=0 GOOS=linux go build -o /GHOPI

#init project
CMD ["/GHOPI"]

