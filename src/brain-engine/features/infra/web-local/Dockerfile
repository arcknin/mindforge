FROM nginx:1.30.5-alpine3.24@sha256:8f84ed99befc3891b8f329c5c202785278a2cfb7c25107d57fb2a134a3117433

RUN rm -f /etc/nginx/conf.d/default.conf

COPY nginx/default.conf /etc/nginx/conf.d/mindforge.conf
COPY build/web/ /usr/share/nginx/html/
