# Use the official Nginx image as the base image
FROM nginx

# Copy your custom Nginx configuration files
COPY default.conf /etc/nginx/conf.d/default.conf
# COPY nginx.conf /etc/nginx/nginx.conf

# Copy SSL certificates
# COPY certificateFiles/cp23kk1.crt /etc/ssl/cp23kk1.crt
# COPY certificateFiles/cp23kk1.key /etc/ssl/cp23kk1.key

# Expose ports
EXPOSE 80
EXPOSE 443

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]