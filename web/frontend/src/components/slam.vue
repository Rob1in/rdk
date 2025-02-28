
<script setup lang="ts">

import { $ref, $computed } from 'vue/macros';
import { grpc } from '@improbable-eng/grpc-web';
import { Client, commonApi, ResponseStream, robotApi, ServiceError, slamApi } from '@viamrobotics/sdk';
import { displayError, isServiceError } from '../lib/error';
import { rcLogConditionally } from '../lib/log';
import PCD from './pcd/pcd-view.vue';
import Slam2dRender from './slam-2d-render.vue';
import { onMounted, onUnmounted } from 'vue';

type MapAndPose = { map: Uint8Array, pose: commonApi.Pose}

const props = defineProps<{
  name: string
  resources: commonApi.ResourceName.AsObject[]
  client: Client
  statusStream: ResponseStream<robotApi.StreamStatusResponse> | null
}>();
const refreshErrorMessage = 'Error refreshing map. The map shown may be stale.';
let refreshErrorMessage2d = $ref<string | null>();
let refreshErrorMessage3d = $ref<string | null>();
let selected2dValue = $ref('manual');
let selected3dValue = $ref('manual');
let pointCloudUpdateCount = $ref(0);
let pointcloud = $ref<Uint8Array | undefined>();
let pose = $ref<commonApi.Pose | undefined>();
let show2d = $ref(false);
let show3d = $ref(false);
let refresh2DCancelled = true;
let refresh3DCancelled = true;

const loaded2d = $computed(() => (pointcloud !== undefined && pose !== undefined));

let slam2dTimeoutId = -1;
let slam3dTimeoutId = -1;

const concatArrayU8 = (arrays: Uint8Array[]) => {
  const totalLength = arrays.reduce((acc, value) => acc + value.length, 0);
  const result = new Uint8Array(totalLength);
  let length = 0;
  for (const array of arrays) {
    result.set(array, length);
    length += array.length;
  }
  return result;
};

const fetchSLAMMap = (name: string): Promise<Uint8Array> => {
  return new Promise((resolve, reject) => {
    const req = new slamApi.GetPointCloudMapRequest();
    req.setName(name);
    rcLogConditionally(req);
    const chunks: Uint8Array[] = [];

    const getPointCloudMap: ResponseStream<slamApi.GetPointCloudMapResponse> =
      props.client.slamService.getPointCloudMap(req);
    getPointCloudMap.on('data', (res: { getPointCloudPcdChunk_asU8(): Uint8Array }) => {
      const chunk = res.getPointCloudPcdChunk_asU8();
      chunks.push(chunk);
    });
    getPointCloudMap.on('status', (status: { code: number, details: string, metadata: grpc.Metadata }) => {
      if (status.code !== 0) {
        const error = {
          message: status.details,
          code: status.code,
          metadata: status.metadata,
        };
        reject(error);
      }
    });
    getPointCloudMap.on('end', (end?: { code: number, details: string, metadata: grpc.Metadata }) => {
      if (end === undefined) {
        const error = { message: 'Stream ended without status code' };
        reject(error);
      } else if (end.code !== 0) {
        const error = {
          message: end.details,
          code: end.code,
          metadata: end.metadata,
        };
        reject(error);
      }
      const arr = concatArrayU8(chunks);
      resolve(arr);
    });
  });
};

const fetchSLAMPose = (name: string): Promise<commonApi.Pose> => {
  return new Promise((resolve, reject): void => {
    const req = new slamApi.GetPositionRequest();
    req.setName(name);
    props.client.slamService.getPosition(
      req,
      new grpc.Metadata(),
      (error: ServiceError | null, res: slamApi.GetPositionResponse | null): void => {
        if (error) {
          reject(error);
          return;
        }
        resolve(res!.getPose()!);
      }
    );
  });
};

const refresh2d = async (name: string) => {
  const map = await fetchSLAMMap(name);
  const returnedPose = await fetchSLAMPose(name);
  const mapAndPose: MapAndPose = {
    map,
    pose: returnedPose,
  };
  return mapAndPose;
};

const handleRefresh2dResponse = (response: MapAndPose): void => {
  pointcloud = response.map;
  pose = response.pose;
  pointCloudUpdateCount += 1;
};

const handleRefresh3dResponse = (response: Uint8Array): void => {
  pointcloud = response;
  pointCloudUpdateCount += 1;
};

const handleError = (errorLocation: string, error: unknown): void => {
  if (isServiceError(error)) {
    displayError(error as ServiceError);
  } else {
    displayError(`${errorLocation} hit error: ${error}`);
  }
};

const scheduleRefresh2d = (name: string, time: string) => {
  const timeoutCallback = async () => {
    try {
      const res = await refresh2d(name);
      handleRefresh2dResponse(res);
    } catch (error) {
      handleError('refresh2d', error);
      selected2dValue = 'manual';
      refreshErrorMessage2d = error !== null && typeof error === 'object' && 'message' in error
        ? `${refreshErrorMessage} ${error.message}`
        : `${refreshErrorMessage} ${error}`;
      return;
    }
    if (refresh2DCancelled) {
      return;
    }
    scheduleRefresh2d(name, time);
  };
  slam2dTimeoutId = window.setTimeout(timeoutCallback, Number.parseFloat(time) * 1000);
};

const scheduleRefresh3d = (name: string, time: string) => {
  const timeoutCallback = async () => {
    try {
      const res = await fetchSLAMMap(name);
      handleRefresh3dResponse(res);
    } catch (error) {
      handleError('fetchSLAMMap', error);
      selected3dValue = 'manual';
      refreshErrorMessage3d = error !== null && typeof error === 'object' && 'message' in error
        ? `${refreshErrorMessage} ${error.message}`
        : `${refreshErrorMessage} ${error}`;
      return;
    }
    if (refresh3DCancelled) {
      return;
    }
    scheduleRefresh3d(name, time);
  };
  slam3dTimeoutId = window.setTimeout(timeoutCallback, Number.parseFloat(time) * 1000);
};

const updateSLAM2dRefreshFrequency = async (name: string, time: 'manual' | string) => {
  refresh2DCancelled = true;
  window.clearTimeout(slam2dTimeoutId);
  refreshErrorMessage2d = null;
  refreshErrorMessage3d = null;

  if (time === 'manual') {
    try {
      const res = await refresh2d(name);
      handleRefresh2dResponse(res);
    } catch (error) {
      handleError('refresh2d', error);
      selected2dValue = 'manual';
      refreshErrorMessage2d = error !== null && typeof error === 'object' && 'message' in error
        ? `${refreshErrorMessage} ${error.message}`
        : `${refreshErrorMessage} ${error}`;
    }
  } else {
    refresh2DCancelled = false;
    scheduleRefresh2d(name, time);
  }
};

const updateSLAM3dRefreshFrequency = async (name: string, time: 'manual' | string) => {
  refresh3DCancelled = true;
  window.clearTimeout(slam3dTimeoutId);
  refreshErrorMessage2d = null;
  refreshErrorMessage3d = null;

  if (time === 'manual') {
    try {
      const res = await fetchSLAMMap(name);
      handleRefresh3dResponse(res);
    } catch (error) {
      handleError('fetchSLAMMap', error);
      selected3dValue = 'manual';
      refreshErrorMessage3d = error !== null && typeof error === 'object' && 'message' in error
        ? `${refreshErrorMessage} ${error.message}`
        : `${refreshErrorMessage} ${error}`;
    }
  } else {
    refresh3DCancelled = false;
    scheduleRefresh3d(name, time);
  }
};

const toggle2dExpand = () => {
  show2d = !show2d;
  if (!show2d) {
    selected2dValue = 'manual';
    return;
  }
  updateSLAM2dRefreshFrequency(props.name, selected2dValue);
};

const toggle3dExpand = () => {
  show3d = !show3d;
  if (!show3d) {
    selected3dValue = 'manual';
    return;
  }
  updateSLAM3dRefreshFrequency(props.name, selected3dValue);
};

const selectSLAM2dRefreshFrequency = () => {
  updateSLAM2dRefreshFrequency(props.name, selected2dValue);
};

const selectSLAMPCDRefreshFrequency = () => {
  updateSLAM3dRefreshFrequency(props.name, selected3dValue);
};

const refresh2dMap = () => {
  updateSLAM2dRefreshFrequency(props.name, 'manual');
};

const refresh3dMap = () => {
  updateSLAM3dRefreshFrequency(props.name, 'manual');
};

onMounted(() => {
  props.statusStream?.on('end', () => {
    window.clearTimeout(slam2dTimeoutId);
    window.clearTimeout(slam3dTimeoutId);
  });
});

onUnmounted(() => {
  window.clearTimeout(slam2dTimeoutId);
  window.clearTimeout(slam3dTimeoutId);
});

</script>

<template>
  <v-collapse
    :title="props.name"
    class="slam"
  >
    <v-breadcrumbs
      slot="title"
      crumbs="slam"
    />
    <div class="border-border-1 h-auto border-x border-b p-2">
      <div class="container mx-auto">
        <div class="flex-col pt-4">
          <div class="flex items-center gap-2">
            <v-switch
              id="showImage"
              :value="show2d ? 'on' : 'off'"
              @input="toggle2dExpand()"
            />
            <span class="pr-2">View SLAM Map (2D)</span>
          </div>
          <div
            v-if="refreshErrorMessage2d && show2d"
            class="border-l-4 border-red-500 bg-gray-100 px-4 py-3"
          >
            {{ refreshErrorMessage2d }}
          </div>
          <div class="float-right pb-4">
            <div class="flex">
              <div
                v-if="show2d"
                class="w-64"
              >
                <p class="font-label mb-1 text-gray-800">
                  Refresh frequency
                </p>
                <div class="relative">
                  <select
                    v-model="selected2dValue"
                    class="
                      border-border-1 m-0 w-full appearance-none border border-solid bg-white
                      bg-clip-padding px-3 py-1.5 text-xs font-normal text-gray-700 focus:outline-none"
                    aria-label="Default select example"
                    @change="selectSLAM2dRefreshFrequency()"
                  >
                    <option
                      value="manual"
                    >
                      Manual
                    </option>
                    <option value="30">
                      Every 30 seconds
                    </option>
                    <option value="10">
                      Every 10 seconds
                    </option>
                    <option value="5">
                      Every 5 seconds
                    </option>
                    <option value="1">
                      Every second
                    </option>
                  </select>
                  <div
                    class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2"
                  >
                    <svg
                      class="h-4 w-4 stroke-2 text-gray-700"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-linejoin="round"
                      stroke-linecap="round"
                      fill="none"
                    >
                      <path d="M18 16L12 22L6 16" />
                    </svg>
                  </div>
                </div>
              </div>
              <div class="px-2 pt-7">
                <v-button
                  v-if="show2d"
                  icon="refresh"
                  label="Refresh"
                  @click="refresh2dMap()"
                />
              </div>
            </div>
          </div>
          <Slam2dRender
            v-if="loaded2d && show2d"
            :point-cloud-update-count="pointCloudUpdateCount"
            :pointcloud="pointcloud"
            :pose="pose"
            :name="name"
            :resources="resources"
            :client="client"
          />
        </div>
        <div class="flex-col pt-4">
          <div class="flex items-center gap-2">
            <v-switch
              :value="show3d ? 'on' : 'off'"
              @input="toggle3dExpand()"
            />
            <span class="pr-2">View SLAM Map (3D)</span>
          </div>
          <div
            v-if="refreshErrorMessage3d && show3d"
            class="border-l-4 border-red-500 bg-gray-100 px-4 py-3"
          >
            {{ refreshErrorMessage3d }}
          </div>
          <div class="float-right pb-4">
            <div class="flex">
              <div
                v-if="show3d"
                class="w-64"
              >
                <p class="font-label mb-1 text-gray-800">
                  Refresh frequency
                </p>
                <div class="relative">
                  <select
                    v-model="selected3dValue"
                    class="
                      border-border-1 m-0 w-full appearance-none border border-solid bg-white
                      bg-clip-padding px-3 py-1.5 text-xs font-normal text-gray-700 focus:outline-none"
                    aria-label="Default select example"
                    @change="selectSLAMPCDRefreshFrequency()"
                  >
                    <option value="manual">
                      Manual
                    </option>
                    <option value="30">
                      Every 30 seconds
                    </option>
                    <option value="10">
                      Every 10 seconds
                    </option>
                    <option value="5">
                      Every 5 seconds
                    </option>
                    <option value="1">
                      Every second
                    </option>
                  </select>
                  <div
                    class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2"
                  >
                    <svg
                      class="h-4 w-4 stroke-2 text-gray-700"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-linejoin="round"
                      stroke-linecap="round"
                      fill="none"
                    >
                      <path d="M18 16L12 22L6 16" />
                    </svg>
                  </div>
                </div>
              </div>
              <div class="px-2 pt-7">
                <v-button
                  v-if="show3d"
                  icon="refresh"
                  label="Refresh"
                  @click="refresh3dMap()"
                />
              </div>
            </div>
          </div>
          <PCD
            v-if="show3d"
            :resources="resources"
            :pointcloud="pointcloud"
            :client="client"
          />
        </div>
      </div>
    </div>
  </v-collapse>
</template>
