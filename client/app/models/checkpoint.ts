//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//
import {Layer} from './layer';
import {Viewport} from '../util/viewport';

/**
 * The types that a value in the Checkpoint could be
 */
export type CheckpointValue = string|number|Viewport|boolean|Layer[]|undefined;

/**
 * Checkpoint stores state related to display options
 */
export declare interface Checkpoint {
  displayOptions: {
    commandFilter: string,
    cpuFilter: string,
    maxIntervalCount: number,
    pidFilter: string,
    tab: string,
    showMigrations: boolean,
    showSleeping: boolean,
    threadListSortField: string,
    threadListSortOrder: number,
    viewport: Viewport,
    layers: Layer[],
    expandedThread: string,
  };
}

