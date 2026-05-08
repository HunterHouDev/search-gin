class VideoClass {
  videoID: string;
  videoElement: HTMLVideoElement;
  constructor(videoID: string) {
    this.videoID = videoID;
    this.videoElement = document.getElementById(videoID) as HTMLVideoElement;
  }
  play() {
    this.videoElement?.play();
  }
  pause() {
    this.videoElement?.pause();
  }
  stop() {
    this.videoElement?.pause();
  }
  forwardTime(n: number) {
    if (this.videoElement) {
      this.videoElement.currentTime = this.videoElement?.currentTime + n;
    }
    return Number(this.videoElement?.currentTime.toFixed(0));
  }
  timeRate(r: number) {
    if (this.videoElement) {
      const newTime = this.videoElement.duration * r;
      // console.log('timeRate', this.videoElement.duration, r, newTime);
      this.videoElement.currentTime = newTime;
    }
    return Number(this.videoElement?.currentTime.toFixed(0));
  }
  timeUpdate(n: number) {
    if (this.videoElement) {
      this.videoElement.currentTime = n;
    }
    return Number(this.videoElement?.currentTime.toFixed(0));
  }
  currentTime() {
    return Number(this.videoElement?.currentTime.toFixed(0));
  }
  duration() {
    return Number(this.videoElement?.duration.toFixed(0));
  }
  volumeUp(n: number) {
    if (this.videoElement) {
      if (this.videoElement.volume + n > 1) {
        this.videoElement.volume = 1;
      } else if (this.videoElement.volume + n < 0) {
        this.videoElement.volume = 0;
      } else {
        this.videoElement.volume = Number(
          (this.videoElement?.volume + n).toFixed(1)
        );
      }
    }
    return Number(this.videoElement?.volume.toFixed(1));
  }

  volumeUpdate(n: number) {
    if (this.videoElement && !isNaN(n)) {
      if (n > 1) {
        n = 1;
      } else if (n < 0) {
        n = 0;
      } else {
        this.videoElement.volume = n;
      }
    }
    return Number(this.videoElement?.volume.toFixed(1));
  }

  async exitPictureInPicture() {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture();
      return;
    }
  }

  async requestPictureInPicture() {
    await this.videoElement?.requestPictureInPicture();
    return;
  }
}

export { VideoClass };
