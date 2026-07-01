import { ref, reactive, onUnmounted } from 'vue';
import { useChatWs } from './useChatWs';
import type { ChatMessage } from './useChatWs';
import { WSMessageType, SignalAction } from 'src/types';

const ICE_SERVERS = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
  ],
};

// 每个浏览器标签页唯一 ID，用于同账号多设备区分
const sessionId = Math.random().toString(36).substring(2, 10);

export function useVideoConference() {
  const { sendJSON, onSignal } = useChatWs();

  const localStream = ref<MediaStream | null>(null);
  const remoteStreams = reactive<Record<string, MediaStream>>({});
  const remoteUsers = ref<string[]>([]);
  const inCall = ref(false);
  const micEnabled = ref(true);
  const camEnabled = ref(true);
  const error = ref<string | null>(null);

  const currentUser = sessionStorage.getItem('username') || '';

  // 存储所有 PeerConnection，key = 对方用户名
  const peers: Record<string, RTCPeerConnection> = {};

  // 清理所有连接
  function closeAllPeers() {
    Object.keys(peers).forEach(key => {
      peers[key].close();
      delete peers[key];
    });
    Object.keys(remoteStreams).forEach(key => {
      delete remoteStreams[key];
    });
    remoteUsers.value = [];
  }

  // 创建一个到对方的 PeerConnection
  function createPeerConnection(remoteUsername: string): RTCPeerConnection {
    if (peers[remoteUsername]) {
      peers[remoteUsername].close();
    }
    const pc = new RTCPeerConnection(ICE_SERVERS);

    // 添加本地流
    if (localStream.value) {
      localStream.value.getTracks().forEach(track => {
        pc.addTrack(track, localStream.value!);
      });
    }

    // 接收远程流
    pc.ontrack = (event) => {
      remoteStreams[remoteUsername] = event.streams[0];
      remoteUsers.value = Object.keys(remoteStreams);
    };

    // ICE candidate → 信令发送
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        sendJSON({
          type: WSMessageType.Signal,
          to: remoteUsername,
          action: SignalAction.Ice,
          data: event.candidate.toJSON(),
        });
      }
    };

    // 连接状态变化
    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed') {
        handleLeave(remoteUsername);
      }
    };

    peers[remoteUsername] = pc;
    return pc;
  }

  // 创建 Offer
  async function createOffer(remoteUsername: string) {
    const pc = createPeerConnection(remoteUsername);
    try {
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      sendJSON({
        type: WSMessageType.Signal,
        to: remoteUsername,
        action: SignalAction.Offer,
        data: { sdp: pc.localDescription?.sdp, type: pc.localDescription?.type },
      });
    } catch (e) {
      console.error('创建 Offer 失败:', e);
    }
  }

  // 处理收到的 Offer
  async function handleOffer(from: string, data: any) {
    const pc = createPeerConnection(from);
    try {
      await pc.setRemoteDescription(new RTCSessionDescription(data));
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      sendJSON({
        type: WSMessageType.Signal,
        to: from,
        action: SignalAction.Answer,
        data: { sdp: pc.localDescription?.sdp, type: pc.localDescription?.type },
      });
    } catch (e) {
      console.error('处理 Offer 失败:', e);
    }
  }

  // 处理收到的 Answer
  async function handleAnswer(from: string, data: any) {
    const pc = peers[from];
    if (!pc) return;
    try {
      await pc.setRemoteDescription(new RTCSessionDescription(data));
    } catch (e) {
      console.error('处理 Answer 失败:', e);
    }
  }

  // 处理 ICE Candidate
  async function handleIce(from: string, data: any) {
    const pc = peers[from];
    if (!pc) return;
    try {
      await pc.addIceCandidate(new RTCIceCandidate(data));
    } catch (e) {
      console.error('添加 ICE Candidate 失败:', e);
    }
  }

  // 处理用户离开
  function handleLeave(username: string) {
    if (peers[username]) {
      peers[username].close();
      delete peers[username];
    }
    delete remoteStreams[username];
    remoteUsers.value = Object.keys(remoteStreams);
  }
  // 注册信号回调
  const unsubSignal = onSignal((msg: ChatMessage) => {
    if (!msg.from || !msg.action) return;

    // 跳过自己发出去的消息（防止同账号多个设备互相干扰）
    if ((msg as any).fromSession === sessionId) return;

    switch (msg.action) {
      case SignalAction.Join:
        if (inCall.value && msg.from !== currentUser) {
          createOffer(msg.from);
        }
        break;
      case SignalAction.Offer:
        if (msg.data) handleOffer(msg.from, msg.data);
        break;
      case SignalAction.Answer:
        if (msg.data) handleAnswer(msg.from, msg.data);
        break;
      case SignalAction.Ice:
        if (msg.data) handleIce(msg.from, msg.data);
        break;
      case SignalAction.Leave:
        handleLeave(msg.from);
        break;
    }
  });

  // 加入会议
  async function join() {
    error.value = null;
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true,
      });
      localStream.value = stream;
      inCall.value = true;

      // 广播给所有人：我进来了
      sendJSON({
        type: WSMessageType.SignalAll,
        action: SignalAction.Join,
        fromSession: sessionId,
      });
    } catch (e: any) {
      error.value = e.message || '无法获取摄像头/麦克风';
      console.error('加入视频会议失败:', e);
    }
  }

  // 离开会议
  function leave() {
    // 广播离开
    sendJSON({
      type: WSMessageType.SignalAll,
      action: SignalAction.Leave,
      fromSession: sessionId,
    });

    // 关闭所有 PeerConnection
    closeAllPeers();

    // 释放本地流
    if (localStream.value) {
      localStream.value.getTracks().forEach(t => t.stop());
      localStream.value = null;
    }

    inCall.value = false;
    error.value = null;
  }

  // 切换麦克风
  function toggleMic() {
    if (localStream.value) {
      localStream.value.getAudioTracks().forEach(t => {
        t.enabled = !t.enabled;
      });
      micEnabled.value = localStream.value.getAudioTracks()[0]?.enabled ?? true;
    }
  }

  // 切换摄像头
  function toggleCam() {
    if (localStream.value) {
      localStream.value.getVideoTracks().forEach(t => {
        t.enabled = !t.enabled;
      });
      camEnabled.value = localStream.value.getVideoTracks()[0]?.enabled ?? true;
    }
  }

  // 组件卸载时自动清理
  onUnmounted(() => {
    unsubSignal();
    if (inCall.value) leave();
  });

  return {
    localStream,
    remoteStreams,
    remoteUsers,
    inCall,
    micEnabled,
    camEnabled,
    error,
    join,
    leave,
    toggleMic,
    toggleCam,
  };
}
