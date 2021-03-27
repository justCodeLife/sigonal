const videoGrid = document.getElementById('video-grid');
const localVideo = document.getElementById('local-video');
const remoteVideo = document.getElementById('remote-video');
let socket
navigator.mediaDevices.getUserMedia({video: true, audio: true})
    .then(stream => {
        videoGrid.style.display = 'grid';
        localVideo.srcObject = stream;
        initConnection(stream);
    }).catch(error => console.log(error));

function initConnection(stream) {
    let localConnection;
    let remoteConnection;
    socket = new WebSocket(`ws://${location.host}/ws`)

    socket.addEventListener('message', (msg) => {
        msg = JSON.parse(msg.data)
        switch (msg.type) {
            case "offer":
                remoteConnection = new RTCPeerConnection();
                stream.getTracks().forEach(track => remoteConnection.addTrack(track, stream))
                remoteConnection.setRemoteDescription(msg).then(() => remoteConnection.createAnswer()).then(ans => {
                    remoteConnection.setLocalDescription(ans).then(() => {
                        socket.send(JSON.stringify({
                            type: 'answer',
                            socket_id: msg.socket_id,
                            sdp: remoteConnection.localDescription.sdp
                        }));
                    })
                })
                remoteConnection.addEventListener('icecandidate', e => {
                    if (e.candidate)
                        socket.send(JSON.stringify({
                            type: 'candidate',
                            socket_id: msg.socket_id,
                            candidate: JSON.stringify(e.candidate)
                        }))
                })
                remoteConnection.addEventListener('track', (e) => {
                    remoteVideo.srcObject = e.streams[0]
                })
                break
            case "answer":
                localConnection.setRemoteDescription(msg).then(() => console.log('remote description set')).catch(err => console.log(err));
                break
            case "candidate":
                const conn = localConnection || remoteConnection;
                conn.addIceCandidate(new RTCIceCandidate(JSON.parse(msg.candidate))).then(() => console.log('ICE CANDIDATE SET')).catch(err => console.log(err));
                break
            case "other-users":
                if (!msg.other_users || !msg.other_users.length) return;
                const socketId = msg.other_users[0];
                localConnection = new RTCPeerConnection();
                stream.getTracks().forEach(track => localConnection.addTrack(track, stream));
                localConnection.addEventListener('icecandidate', (e) => {
                    if (e.candidate) {
                        socket.send(JSON.stringify({
                            type: 'candidate',
                            socket_id: socketId,
                            candidate: JSON.stringify(e.candidate)
                        }));
                    }
                })
                localConnection.addEventListener('track', (e) => {
                    remoteVideo.srcObject = e.streams[0]
                })
                localConnection.createOffer().then(offer => localConnection.setLocalDescription(offer)).then(() => {
                    socket.send(JSON.stringify({
                        type: 'offer',
                        socket_id: socketId,
                        sdp: localConnection.localDescription.sdp
                    }));
                });
                break
            default:
                console.log('Invalid Message')
                break
        }
    })
    socket.addEventListener('error', () => {
        console.log('error happened')
        socket.close()
    })
    socket.addEventListener('close', () => {
        console.log('closed')
        setTimeout(function () {
            initConnection(stream)
        }, 1000);
    })
}
