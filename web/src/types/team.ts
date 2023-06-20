/*
Licensed to LinDB under one or more contributor
license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright
ownership. LinDB licenses this file to you under
the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
 
Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/
export interface TeamResult {
  teams?: Team[];
  total?: number;
}

export interface Team {
  uid?: string;
  name?: string;
  email?: string;
}

export interface SearchTeam {
  name?: string;
}

export interface TeamMemberResult {
  members?: TeamMember[];
  total?: number;
}

export interface TeamMember {
  userUid: string;
  userName: string;
  name: string;
  permission: string;
}

export interface SearchTeamMember {
  user?: string;
  permissions?: string[];
}

export interface AddTeamMembers {
  userUids: string[];
  permission: string;
}

export interface UpdateTeamMember {
  userUid: string;
  permission: string;
}

export interface RemoveTeamMembers {
  userUids: string[];
}
