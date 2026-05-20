class VideoClass {
  videoID: string;
  constructor(videoID: string) {
    this.videoID = videoID;
  }
  private getVideoElement(): HTMLVideoElement | null {
    return document.getElementById(this.videoID) as HTMLVideoElement;
  }
  play() {
    this.getVideoElement()?.play();
  }
  pause() {
    this.getVideoElement()?.pause();
  }
  stop() {
    this.getVideoElement()?.pause();
  }
  forwardTime(n: number) {
    const el = this.getVideoElement();
    if (el) {
      el.currentTime = el.currentTime + n;
    }
    return Number(el?.currentTime.toFixed(0));
  }
  timeRate(r: number) {
    const el = this.getVideoElement();
    if (el) {
      const newTime = el.duration * r;
      el.currentTime = newTime;
    }
    return Number(el?.currentTime.toFixed(0));
  }
  timeUpdate(n: number) {
    const el = this.getVideoElement();
    if (el) {
      el.currentTime = n;
    }
    return Number(el?.currentTime.toFixed(0));
  }
  currentTime() {
    const el = this.getVideoElement();
    return Number(el?.currentTime.toFixed(0));
  }
  duration() {
    const el = this.getVideoElement();
    return Number(el?.duration.toFixed(0));
  }
  volumeUp(n: number) {
    const el = this.getVideoElement();
    if (el) {
      if (el.volume + n > 1) {
        el.volume = 1;
      } else if (el.volume + n < 0) {
        el.volume = 0;
      } else {
        el.volume = Number((el.volume + n).toFixed(1));
      }
    }
    return Number(el?.volume.toFixed(1));
  }
  volumeUpdate(n: number) {
    const el = this.getVideoElement();
    if (el && !isNaN(n)) {
      if (n > 1) {
        n = 1;
      } else if (n < 0) {
        n = 0;
      } else {
        el.volume = n;
      }
    }
    return Number(el?.volume.toFixed(1));
  }

  async exitPictureInPicture() {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture();
      return;
    }
  }

  async requestPictureInPicture() {
    await this.getVideoElement()?.requestPictureInPicture();
    return;
  }
}

export { VideoClass };
