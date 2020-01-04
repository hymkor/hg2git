if os.getenv("LANG") ~= "C" then
    if string.match(os.getenv("OS") or "","^Windows") then
        os.execute('set "LANG=C" & lua "' .. arg[0] .. '"')
    else
        os.execute('LANG=C ; export LANG ; lua "' .. arg[0] .. '"')
    end
    os.exit(0)
end

local hgRep = {}
local hgCnt = 0
local fd = io.popen("hg log -v","r")
if fd then
    local cur
    local array = {}
    for line in fd:lines() do
        local serial,id = string.match(line,"^changeset:%s*([0-9]+):(.*)$")
        if serial then
            cur = { id=id }
            array[ 0+serial ] = cur
            hgRep[ id ] = cur
            hgCnt = hgCnt + 1
        else
            id = string.match(line,"^parent:%s*[0-9]+:(.*)")
            if id and cur then
                local p = (cur.parents or {})
                p[id] = true
                cur.parents = p
            end
        end
    end
    fd:close()
    for key,val in pairs(array) do
        if not val.parents and array[key-1] then
            val.parents = { [array[key-1].id]=true }
        end
    end
end

local branch = {}
fd = io.popen("git branch","r")
if fd then
    for line in fd:lines() do
        branch[#branch+1] = string.sub(line,3)
    end
    fd:close()
end

local gitCnt = 0
local gitRep = {}
local short2long = {}
for _,branch1 in pairs(branch) do
    local cmdline = "git log -v "..branch1 
    print(cmdline)
    fd = io.popen(cmdline,"r")
    if fd then
        local cur = {}
        for line in fd:lines() do
            local id = string.match(line,"^commit%s+(.*)$")
            if id then
                if cur.parents then
                    cur.parents[id] = true 
                else
                    cur.parents = { [id] = true }
                end
                cur = gitRep[ id ] or { id=id }
                gitRep[ id ] = cur
                gitCnt = gitCnt + 1
                short2long[ string.sub(id,1,7) ] = id
            else
                local ids = string.match(line,"^Merge:%s+(.*)$")
                if ids then
                    for id1 in string.gmatch(ids,"%S+") do
                        if cur.parents then
                            cur.parents[id1] = true
                        else
                            cur.parents = { [id1] = true }
                        end
                    end
                else
                    local hgid = string.match(line,"^%s*HG:%s*(.*)$")
                    if hgid then
                        cur.hg = hgid
                        assert(hgRep[ hgid ]).git = cur
                    end
                end
            end
        end
        fd:close()
    end
end

print("sizeof( hg commit)=",hgCnt)
print("sizeof(git commit)=",gitCnt)

function getGitCom(id)
    return gitRep[id] or assert(gitRep[ assert(short2long[id],(id or "(null)").. "not found(short2long") ],
                id .. "not found ?(gitRep)")
end

function has_parent(gitCom,hg_par_id)
    for par_id,_ in pairs(gitCom.parents or {}) do
        if getGitCom(par_id).hg == hg_par_id then
            return true
        end
    end
    return false
end

for _,hgCom in pairs(hgRep) do
    if not hgCom.git then
        print(string.formart("HG %s does not has GIT commit",hg.id))
    else
        for hg_par_id,_ in pairs(hgCom.parents or {}) do
            if not has_parent(hgCom.git , hg_par_id) then
                print(string.format("HG:[%s] / git:[%s]",hgCom.id, hgCom.git.id))
                print(string.format("  has no parent: HG:[%s]",hg_par_id))

                for git_par_id,_ in pairs(getGitCom(hgCom.git.id).parents or {}) do
                    print(string.format("  git-parent: %s",git_par_id))
                    print(string.format("  hg-parent: %s",getGitCom(git_par_id).hg))
                end
            end
        end
    end
end
