import type {
  dictservicev1_DeleteDictTypeRequest,
  dictservicev1_GetDictTypeRequest,
  dictservicev1_ListDictTypeResponse,
  dictservicev1_DictType,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// DictType 管理
// ==============================

export function useListDictTypes(
  query: PaginationQuery,
  options?: UseQueryOptions<dictservicev1_ListDictTypeResponse, Error>,
) {
  return useQuery({
    queryKey: ['listDictTypes', query],
    queryFn: () => apiClient.dictTypeService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListDictTypes(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listDictTypes', params],
    queryFn: () => apiClient.dictTypeService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetDictType(
  req: dictservicev1_GetDictTypeRequest,
  options?: UseQueryOptions<dictservicev1_DictType, Error>,
) {
  return useQuery({
    queryKey: ['getDictType', req],
    queryFn: () => apiClient.dictTypeService.Get(req),
    ...options,
  });
}

export function useCreateDictType(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.dictTypeService.Create({ data: { ...values } as dictservicev1_DictType }),
    ...options,
  });
}

export function useUpdateDictType(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.dictTypeService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteDictType(
  options?: UseMutationOptions<
    object,
    Error,
    dictservicev1_DeleteDictTypeRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.dictTypeService.Delete(req),
    ...options,
  });
}
