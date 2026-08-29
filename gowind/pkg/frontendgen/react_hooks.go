package frontendgen

import "strings"

// reactHooksCode 生成 api/hooks/*.ts（对应 TS 版 hooks-template.ts）
// 直接使用 @/api/client 的 apiClient 单例调用 Service Client。
func reactHooksCode(service *ParsedService, serviceName string) string {
	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	hasGet := crudPaths.Get != nil
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil
	hasDelete := crudPaths.Delete != nil

	modelPascal := toPascalCase(service.ModelName)
	prefix := service.TypePrefix
	client := "apiClient." + service.ClientGetterName

	var typeImports []string
	if hasCreate {
		typeImports = append(typeImports, "type "+prefix+"_Create"+modelPascal+"Request")
	}
	if hasDelete {
		typeImports = append(typeImports, "type "+prefix+"_Delete"+modelPascal+"Request")
	}
	if hasGet {
		typeImports = append(typeImports, "type "+prefix+"_Get"+modelPascal+"Request")
	}
	if hasList {
		typeImports = append(typeImports, "type "+prefix+"_List"+modelPascal+"Response")
	}
	if hasGet || hasList {
		typeImports = append(typeImports, "type "+prefix+"_"+modelPascal)
	}

	var sb strings.Builder
	sb.WriteString(`import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
`)

	if len(typeImports) > 0 {
		sb.WriteString("import {\n")
		for _, imp := range typeImports {
			sb.WriteString("  " + imp + ",\n")
		}
		sb.WriteString("} from '@/api/generated/" + serviceName + "/service/v1';\n")
	}

	sb.WriteString(`import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

`)

	sb.WriteString("// ==============================\n")
	sb.WriteString("// " + service.ModelName + " 管理\n")
	sb.WriteString("// ==============================\n\n")

	if hasList {
		sb.WriteString(`export function useList` + modelPascal + `s(
  query: PaginationQuery,
  options?: UseQueryOptions<` + prefix + `_List` + modelPascal + `Response, Error>,
) {
  return useQuery({
    queryKey: ['list` + modelPascal + `s', query],
    queryFn: () => ` + client + `.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchList` + modelPascal + `s(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['list` + modelPascal + `s', params],
    queryFn: () => ` + client + `.List(params.toRawParams()),
    retry: 0,
  });
}

`)
	}

	if hasGet {
		sb.WriteString(`export function useGet` + modelPascal + `(
  req: ` + prefix + `_Get` + modelPascal + `Request,
  options?: UseQueryOptions<` + prefix + `_` + modelPascal + `, Error>,
) {
  return useQuery({
    queryKey: ['get` + modelPascal + `', req],
    queryFn: () => ` + client + `.Get(req),
    ...options,
  });
}

`)
	}

	if hasCreate {
		sb.WriteString(`export function useCreate` + modelPascal + `(
  options?: UseMutationOptions<{}, Error, ` + prefix + `_Create` + modelPascal + `Request>,
) {
  return useMutation({
    mutationFn: (data) => ` + client + `.Create(data),
    ...options,
  });
}

`)
	}

	if hasUpdate {
		sb.WriteString(`export function useUpdate` + modelPascal + `(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      ` + client + `.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

`)
	}

	if hasDelete {
		sb.WriteString(`export function useDelete` + modelPascal + `(
  options?: UseMutationOptions<{}, Error, ` + prefix + `_Delete` + modelPascal + `Request>,
) {
  return useMutation({
    mutationFn: (req) => ` + client + `.Delete(req),
    ...options,
  });
}
`)
	}

	return sb.String()
}
