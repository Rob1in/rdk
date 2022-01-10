const { grpc } = require("@improbable-eng/grpc-web");
window.robotApi = require('./gen/proto/api/v1/robot_pb.js');
const { RobotServiceClient } = require('./gen/proto/api/v1/robot_pb_service.js');
window.metadataApi = require('./gen/proto/api/service/v1/metadata_pb.js');
const { MetadataServiceClient } = require('./gen/proto/api/service/v1/metadata_pb_service.js');
window.forceMatrixApi = require('./gen/proto/api/component/v1/forcematrix_pb.js');
const { ForceMatrixServiceClient } = require('./gen/proto/api/component/v1/forcematrix_pb_service.js');
window.armApi = require('./gen/proto/api/component/v1/arm_pb.js');
const { ArmServiceClient } = require('./gen/proto/api/component/v1/arm_pb_service.js');
window.gantryApi = require('./gen/proto/api/component/v1/gantry_pb.js');
const { GantryServiceClient } = require('./gen/proto/api/component/v1/gantry_pb_service.js');
window.gripperApi = require('./gen/proto/api/component/v1/gripper_pb.js');
const { GripperServiceClient } = require('./gen/proto/api/component/v1/gripper_pb_service.js');
window.servoApi = require('./gen/proto/api/component/v1/servo_pb.js');
const { ServoServiceClient } = require('./gen/proto/api/component/v1/servo_pb_service.js');
window.motorApi = require('./gen/proto/api/component/v1/motor_pb.js');
const { MotorServiceClient } = require('./gen/proto/api/component/v1/motor_pb_service.js');
window.cameraApi = require('./gen/proto/api/component/v1/camera_pb.js');
const { CameraServiceClient } = require('./gen/proto/api/component/v1/camera_pb_service.js');
window.inputApi = require('./gen/proto/api/component/v1/input_controller_pb.js');
const { InputControllerServiceClient } = require('./gen/proto/api/component/v1/input_controller_pb_service.js');
window.commonApi = require('./gen/proto/api/common/v1/common_pb.js');
const { StreamServiceClient } = require('./gen/proto/stream/v1/stream_pb_service.js');
window.streamApi = require("./gen/proto/stream/v1/stream_pb.js");
const { dialDirect, dialWebRTC } = require("@viamrobotics/rpc");
window.THREE = require("three/build/three.module.js")
window.pcdLib = require("three/examples/jsm/loaders/PCDLoader.js")
window.orbitLib = require("three/examples/jsm/controls/OrbitControls.js")
window.trackLib = require("three/examples/jsm/controls/TrackballControls.js")
const rtcConfig = {
	iceServers: [
		{
			urls: 'stun:global.stun.twilio.com:3478?transport=udp'
		}
	]
}

if (window.webrtcAdditionalICEServers) {
	rtcConfig.iceServers = rtcConfig.iceServers.concat(window.webrtcAdditionalICEServers);
}

let connect = async (creds) => {
	let transportFactory;
	const opts = { credentials: creds, webrtcOptions: { rtcConfig: rtcConfig } };
	if (window.webrtcEnabled) {
		const webRTCConn = await dialWebRTC(window.webrtcSignalingAddress, window.webrtcHost, opts);
		transportFactory = webRTCConn.transportFactory
		window.streamService = new StreamServiceClient(window.webrtcHost, { transport: transportFactory });
		webRTCConn.peerConnection.ontrack = async event => {
			const video = document.createElement('video');
			video.srcObject = event.streams[0];
			video.autoplay = true;
			video.controls = false;
			video.playsInline = true;
			const streamName = event.streams[0].id;
			const streamContainer = document.getElementById(`stream-${streamName}`);
			streamContainer.getElementsByTagName("button")[0].remove();
			streamContainer.appendChild(video);
		}
	} else {
		const url = `${location.protocol}//${location.hostname}${location.port ? ':' + location.port : ''}`;
		transportFactory = await dialDirect(url, opts);
	}

	window.connect = () => connect(creds); // save creds

	window.robotService = new RobotServiceClient(window.webrtcHost, { transport: transportFactory });
	window.metadataService = new MetadataServiceClient(window.webrtcHost, { transport: transportFactory });

	// TODO: these should be created as needed for #272
	window.armService = new ArmServiceClient(window.webrtcHost, { transport: transportFactory });
	window.forceMatrixService = new ForceMatrixServiceClient(window.webrtcHost, { transport: transportFactory });
	window.gantryService = new GantryServiceClient(window.webrtcHost, { transport: transportFactory });
	window.gripperService = new GripperServiceClient(window.webrtcHost, { transport: transportFactory });
	window.servoService = new ServoServiceClient(window.webrtcHost, { transport: transportFactory });
	window.cameraService = new CameraServiceClient(window.webrtcHost, { transport: transportFactory });
	window.inputControllerService = new InputControllerServiceClient(window.webrtcHost, { transport: transportFactory });
	window.motorService = new MotorServiceClient(window.webrtcHost, { transport: transportFactory });
}
window.connect = connect;
