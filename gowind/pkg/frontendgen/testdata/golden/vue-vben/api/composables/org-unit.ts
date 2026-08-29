import type {
  identityservicev1_DeleteOrgUnitRequest,
  identityservicev1_GetOrgUnitRequest,
  identityservicev1_ListOrgUnitResponse,
  identityservicev1_OrgUnit,
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
// OrgUnit 管理
// ==============================

export function useListOrgUnits(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListOrgUnitResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOrgUnits', query],
    queryFn: () => apiClient.orgUnitService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOrgUnits(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOrgUnits', params],
    queryFn: () => apiClient.orgUnitService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetOrgUnit(
  req: identityservicev1_GetOrgUnitRequest,
  options?: UseQueryOptions<identityservicev1_OrgUnit, Error>,
) {
  return useQuery({
    queryKey: ['getOrgUnit', req],
    queryFn: () => apiClient.orgUnitService.Get(req),
    ...options,
  });
}

export function useCreateOrgUnit(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.orgUnitService.Create({ data: { ...values } as identityservicev1_OrgUnit }),
    ...options,
  });
}

export function useUpdateOrgUnit(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.orgUnitService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteOrgUnit(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeleteOrgUnitRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.orgUnitService.Delete(req),
    ...options,
  });
}
