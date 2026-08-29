import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type dictservicev1_CreateDictTypeRequest,
  type dictservicev1_DeleteDictTypeRequest,
  type dictservicev1_GetDictTypeRequest,
  type dictservicev1_ListDictTypeResponse,
  type dictservicev1_DictType,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

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
  options?: UseMutationOptions<{}, Error, dictservicev1_CreateDictTypeRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.dictTypeService.Create(data),
    ...options,
  });
}

export function useUpdateDictType(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
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
  options?: UseMutationOptions<{}, Error, dictservicev1_DeleteDictTypeRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.dictTypeService.Delete(req),
    ...options,
  });
}
