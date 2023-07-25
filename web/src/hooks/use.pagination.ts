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
import { parseInt } from 'lodash-es';
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';

const usePagination = (initPage: number = 1, initPageSize: number = 20) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [currentPage, setCurrentPage] = useState(() => {
    if (searchParams.has('currentPage')) {
      return parseInt(`${searchParams.get('currentPage')}`);
    }
    return initPage;
  });

  const [pageSize, setPageSize] = useState(() => {
    if (searchParams.has('pageSize')) {
      return parseInt(`${searchParams.get('pageSize')}`);
    }
    return initPageSize;
  });

  const onChange = (page: number, pageSize: number) => {
    setCurrentPage(page);
    setPageSize(pageSize);
    searchParams.set('currentPage', `${page}`);
    searchParams.set('pageSize', `${pageSize}`);
    setSearchParams(searchParams);
  };

  return {
    currentPage,
    pageSize,
    offset: (currentPage - 1) * pageSize,
    onChange,
  };
};

export { usePagination };
