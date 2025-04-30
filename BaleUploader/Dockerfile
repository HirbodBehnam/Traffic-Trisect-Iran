# Based on https://github.com/iphoneintosh/ubuntu-docker
FROM debian:12

# prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# update dependencies
RUN apt update
RUN apt upgrade -y

# install xfce desktop
RUN apt install -y xfce4 xfce4-goodies
# This causes OOM somehow. Remove it
RUN apt purge -y xfce4-power-manager

# install dependencies
RUN apt install -y \
  tightvncserver \
  net-tools \
  nano \
  vim \
  curl \
  wget \
  chromium \
  chromium-driver \
  python3 \
  python3-pip \
  python3-full \
  sudo \
  dbus-x11 \
  iproute2

# xfce fixes
RUN update-alternatives --set x-terminal-emulator /usr/bin/xfce4-terminal.wrapper

# Python packages
RUN pip3 install --break-system-packages selenium

# User settings
RUN useradd -m -s /bin/bash hirbod
RUN usermod -aG sudo hirbod
RUN echo 'hirbod:123456' | chpasswd
USER hirbod
ENV USER=hirbod

COPY uploader.py /home/hirbod/uploader.py

# VNC and noVNC config
ARG VNCPORT=5900
ENV VNCPORT=${VNCPORT}
EXPOSE ${VNCPORT}

ARG VNCPWD=changeme
ENV VNCPWD=${VNCPWD}

ARG VNCDISPLAY=1024x768
ENV VNCDISPLAY=${VNCDISPLAY}

ARG VNCDEPTH=16
ENV VNCDEPTH=${VNCDEPTH}

# setup VNC
RUN mkdir -p $HOME/.vnc/
RUN echo ${VNCPWD} | vncpasswd -f > $HOME/.vnc/passwd
RUN chmod 600 $HOME/.vnc/passwd
RUN echo "#!/bin/sh \n\
xrdb $HOME/.Xresources \n\
xsetroot -solid grey \n\
# Fix to make GNOME work \n\
export XKL_XMODMAP_DISABLE=1 \n\
/etc/X11/Xsession \n\
startxfce4 & \n\
" > $HOME/.vnc/xstartup
RUN chmod +x $HOME/.vnc/xstartup

ENTRYPOINT [ "/bin/bash", "-c", " \
  vncserver :0 -rfbport ${VNCPORT} -geometry $VNCDISPLAY -depth $VNCDEPTH; \
  bash; \
" ]