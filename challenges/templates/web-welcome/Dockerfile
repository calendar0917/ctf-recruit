FROM busybox:1.36
WORKDIR /www
COPY index.html /www/index.html
EXPOSE 80
CMD ["httpd", "-f", "-p", "80", "-h", "/www"]
